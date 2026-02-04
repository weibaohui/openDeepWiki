#!/usr/bin/env python3
"""
Go 后端技术栈初步扫描脚本

功能：
1) 解析仓库内所有 go.mod，收集 module / go 版本 / require / replace；
2) 扫描 .go 文件的 import 与关键调用点；
3) 基于启发式规则识别常见技术栈，并输出带行号证据的 JSON。

注意：
- 该脚本的目标是“快速初筛 + 给出证据位置”，不是完整的静态分析器。
- 对于只在测试代码中出现的依赖，会降低置信度。
"""

from __future__ import annotations

import argparse
import json
import os
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence, Set, Tuple


@dataclass(frozen=True)
class Evidence:
    """技术栈判定证据（包含文件与行号）。"""

    file: str
    line_start: int
    line_end: int
    match: str
    why: str


@dataclass(frozen=True)
class StackRule:
    """技术栈识别规则。"""

    category: str
    name: str
    import_markers: Tuple[str, ...]
    code_markers: Tuple[str, ...]


def _is_ignored_dir(dir_name: str) -> bool:
    """判断目录是否应跳过（减少误判与加速扫描）。"""

    return dir_name in {
        ".git",
        "vendor",
        "node_modules",
        ".idea",
        ".vscode",
        ".trae",
        "dist",
        "build",
        "tmp",
        "bin",
        "out",
    }


def _walk_files(root: Path, suffix: str) -> Iterable[Path]:
    """遍历 root 下指定后缀的文件，跳过常见无关目录。"""

    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if not _is_ignored_dir(d)]
        for filename in filenames:
            if filename.endswith(suffix):
                yield Path(dirpath) / filename


def _walk_all_files(root: Path) -> Iterable[Path]:
    """遍历 root 下所有文件，跳过常见无关目录。"""

    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if not _is_ignored_dir(d)]
        for filename in filenames:
            yield Path(dirpath) / filename


def _read_text(path: Path) -> str:
    """以尽可能宽容的方式读取文本文件。"""

    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return path.read_text(encoding="utf-8", errors="replace")


_GO_MOD_MODULE_RE = re.compile(r"^\s*module\s+(\S+)\s*$", re.MULTILINE)
_GO_MOD_GO_RE = re.compile(r"^\s*go\s+(\S+)\s*$", re.MULTILINE)
_GO_MOD_TOOLCHAIN_RE = re.compile(r"^\s*toolchain\s+(\S+)\s*$", re.MULTILINE)


def _parse_go_mod(content: str) -> Tuple[Optional[str], Optional[str], Optional[str], List[str], List[str]]:
    """解析 go.mod 的核心信息：module/go/toolchain/require/replace。"""

    module = None
    go_version = None
    toolchain = None

    m = _GO_MOD_MODULE_RE.search(content)
    if m:
        module = m.group(1).strip()

    m = _GO_MOD_GO_RE.search(content)
    if m:
        go_version = m.group(1).strip()

    m = _GO_MOD_TOOLCHAIN_RE.search(content)
    if m:
        toolchain = m.group(1).strip()

    requires: List[str] = []
    replaces: List[str] = []

    in_require_block = False
    in_replace_block = False

    for raw_line in content.splitlines():
        line = raw_line.strip()
        if not line:
            continue

        if "//" in line:
            line = line.split("//", 1)[0].strip()
        if not line:
            continue

        if line == ")":
            in_require_block = False
            in_replace_block = False
            continue

        if line.startswith("require ("):
            in_require_block = True
            continue
        if line.startswith("replace ("):
            in_replace_block = True
            continue

        if in_require_block:
            parts = line.split()
            if len(parts) >= 2:
                requires.append(f"{parts[0]}@{parts[1]}")
            continue

        if in_replace_block:
            replaces.append(line)
            continue

        if line.startswith("require "):
            parts = line[len("require ") :].strip().split()
            if len(parts) >= 2:
                requires.append(f"{parts[0]}@{parts[1]}")
            else:
                requires.append(" ".join(parts))
            continue

        if line.startswith("replace "):
            replaces.append(line[len("replace ") :].strip())
            continue

    return module, go_version, toolchain, requires, replaces


def collect_go_mods(repo_root: Path) -> List[Dict]:
    """收集仓库中的 go.mod 信息。"""

    mods: List[Dict] = []
    for go_mod_path in _walk_files(repo_root, "go.mod"):
        content = _read_text(go_mod_path)
        module, go_version, toolchain, requires, replaces = _parse_go_mod(content)
        mods.append(
            {
                "path": str(go_mod_path.relative_to(repo_root)),
                "module": module,
                "go_version": go_version,
                "toolchain": toolchain,
                "requires": requires,
                "replaces": replaces,
            }
        )
    mods.sort(key=lambda x: x["path"])
    return mods


_IMPORT_BLOCK_RE = re.compile(r"(?s)\bimport\s*\((.*?)\)\s")
_IMPORT_LINE_RE = re.compile(r'^\s*import\s+"([^"]+)"\s*$', re.MULTILINE)
_IMPORT_QUOTED_RE = re.compile(r'^\s*(?:\w+\s+)?"([^"]+)"\s*$', re.MULTILINE)


def extract_imports(go_source: str) -> Set[str]:
    """从 Go 源码中粗略提取 import 路径集合。"""

    imports: Set[str] = set()
    for m in _IMPORT_LINE_RE.finditer(go_source):
        imports.add(m.group(1))

    for m in _IMPORT_BLOCK_RE.finditer(go_source):
        block = m.group(1)
        for m2 in _IMPORT_QUOTED_RE.finditer(block):
            imports.add(m2.group(1))

    return imports


def is_test_file(path: Path) -> bool:
    """判断文件是否为测试文件或测试目录下的文件。"""

    if path.name.endswith("_test.go"):
        return True
    parts = [p.lower() for p in path.parts]
    return "test" in parts or "tests" in parts


def find_marker_evidence(
    repo_root: Path,
    marker: str,
    max_hits: int,
    ignore_tests: bool,
) -> List[Evidence]:
    """在仓库中查找 marker 的出现位置，并输出证据。"""

    hits: List[Evidence] = []
    for go_file in _walk_files(repo_root, ".go"):
        if ignore_tests and is_test_file(go_file):
            continue
        text = _read_text(go_file)
        if marker not in text:
            continue
        for idx, line in enumerate(text.splitlines(), start=1):
            if marker in line:
                hits.append(
                    Evidence(
                        file=str(go_file.relative_to(repo_root)),
                        line_start=idx,
                        line_end=idx,
                        match=line.strip()[:200],
                        why=f"命中关键特征：{marker}",
                    )
                )
                if len(hits) >= max_hits:
                    return hits
    return hits


def find_marker_evidence_in_files(
    repo_root: Path,
    relative_files: Sequence[str],
    marker: str,
    max_hits: int,
    why: str,
) -> List[Evidence]:
    """在指定文件集合中查找 marker 的出现位置，并输出证据（包含行号）。"""

    hits: List[Evidence] = []
    for rel in relative_files:
        p = (repo_root / rel).resolve()
        if not p.exists() or not p.is_file():
            continue
        text = _read_text(p)
        if marker not in text:
            continue
        for idx, line in enumerate(text.splitlines(), start=1):
            if marker in line:
                hits.append(
                    Evidence(
                        file=str(Path(rel)),
                        line_start=idx,
                        line_end=idx,
                        match=line.strip()[:200],
                        why=why,
                    )
                )
                if len(hits) >= max_hits:
                    return hits
    return hits


def detect_stacks(
    repo_root: Path,
    go_mods: Sequence[Dict],
    max_evidence: int,
    ignore_tests: bool,
    verbose: bool,
) -> List[Dict]:
    """根据启发式规则识别技术栈，返回结构化结果。"""

    rules: List[StackRule] = [
        StackRule(
            "ai_orchestration",
            "eino",
            ("github.com/cloudwego/eino",),
            (
                "adk.NewChatModelAgent(",
                "adk.NewSequentialAgent(",
                "adk.NewRunner(",
                "compose.ToolsNodeConfig",
                "callbacks.AppendGlobalHandlers(",
            ),
        ),
        StackRule(
            "ai_llm",
            "openai(eino-ext)",
            ("github.com/cloudwego/eino-ext/components/model/openai",),
            ("openai.NewChatModel(", "openai.ChatModelConfig{"),
        ),
        StackRule(
            "ai_llm",
            "openai-compatible(chat-completions)",
            tuple(),
            (
                "/chat/completions",
                'json:"tool_choice,omitempty"',
                'json:"tool_calls,omitempty"',
                'Role:       "tool"',
            ),
        ),
        StackRule("web_framework", "gin", ("github.com/gin-gonic/gin",), ("gin.Default(", "gin.New(")),
        StackRule("web_framework", "echo", ("github.com/labstack/echo/v4",), ("echo.New(",)),
        StackRule("web_framework", "fiber", ("github.com/gofiber/fiber/v2",), ("fiber.New(", "app.Listen(")),
        StackRule("web_framework", "chi", ("github.com/go-chi/chi/v5",), ("chi.NewRouter(",)),
        StackRule("api_style", "grpc", ("google.golang.org/grpc",), ("grpc.NewServer(", "grpc.Dial(", "grpc.DialContext(")),
        StackRule("api_style", "grpc-gateway", ("github.com/grpc-ecosystem/grpc-gateway",), ("runtime.NewServeMux(", "HandlerFromEndpoint(")),
        StackRule("api_style", "graphql(gqlgen)", ("github.com/99designs/gqlgen",), ("NewExecutableSchema(", "NewDefaultServer(")),
        StackRule("database", "gorm", ("gorm.io/gorm",), ("gorm.Open(", "AutoMigrate(")),
        StackRule("database", "sqlx", ("github.com/jmoiron/sqlx",), ("sqlx.Connect(", "sqlx.Open(")),
        StackRule("database", "mongodb", ("go.mongodb.org/mongo-driver/mongo",), ("mongo.Connect(",)),
        StackRule(
            "cache_kv",
            "redis(go-redis)",
            ("github.com/redis/go-redis/v9", "github.com/go-redis/redis/v8"),
            ("redis.NewClient(", "redis.NewClusterClient(", "redis.ParseURL("),
        ),
        StackRule("task_job", "cron(robfig)", ("github.com/robfig/cron/v3",), ("cron.New(", "AddFunc(")),
        StackRule("task_job", "asynq", ("github.com/hibiken/asynq",), ("asynq.NewServer(", "asynq.NewClient(", "asynq.NewScheduler(")),
        StackRule("task_job", "temporal", ("go.temporal.io/sdk",), ("worker.New(", "ExecuteActivity(")),
        StackRule("message_queue", "kafka(segmentio)", ("github.com/segmentio/kafka-go",), ("kafka.NewReader(", "kafka.Writer")),
        StackRule("message_queue", "nats", ("github.com/nats-io/nats.go",), ("nats.Connect(", "Subscribe(")),
        StackRule("message_queue", "rabbitmq(amqp)", ("github.com/rabbitmq/amqp091-go", "github.com/streadway/amqp"), ("amqp.Dial(", "Consume(")),
        StackRule("config", "viper", ("github.com/spf13/viper",), ("viper.ReadInConfig(", "viper.Unmarshal(", "viper.SetConfigFile(")),
        StackRule("auth_security", "jwt", ("github.com/golang-jwt/jwt/v5",), ("jwt.Parse(", "NewWithClaims(")),
        StackRule("auth_security", "casbin", ("github.com/casbin/casbin/v2",), ("casbin.NewEnforcer(",)),
        StackRule("observability", "opentelemetry", ("go.opentelemetry.io/otel",), ("otel.Tracer(", "SpanFromContext(")),
        StackRule("observability", "prometheus", ("github.com/prometheus/client_golang/prometheus",), ("promhttp.Handler(", "prometheus.NewRegistry(")),
        StackRule("observability", "klog", ("k8s.io/klog/v2",), ("klog.V(", "klog.Info", "klog.Error")),
        StackRule("observability", "zap", ("go.uber.org/zap",), ("zap.New(", "zap.L(", ".With(")),
        StackRule("observability", "logrus", ("github.com/sirupsen/logrus",), ("logrus.New(", "logrus.WithField(", "logrus.WithFields(")),
    ]

    requires_flat: List[str] = []
    for m in go_mods:
        requires_flat.extend(m.get("requires", []) or [])
    go_mod_files = [m.get("path") for m in go_mods if m.get("path")]

    def in_go_mod(marker: str) -> bool:
        return any(marker in req for req in requires_flat)

    go_imports: Set[str] = set()
    for go_file in _walk_files(repo_root, ".go"):
        if ignore_tests and is_test_file(go_file):
            continue
        go_imports |= extract_imports(_read_text(go_file))

    def in_imports(marker: str) -> bool:
        return any(imp == marker or imp.startswith(marker + "/") for imp in go_imports)

    stacks: List[Dict] = []

    for rule in rules:
        matched_by_mod = any(in_go_mod(m) for m in rule.import_markers)
        matched_by_import = any(in_imports(m) for m in rule.import_markers)

        matched_markers: List[str] = []
        for m in rule.import_markers:
            if in_go_mod(m) or in_imports(m):
                matched_markers.append(m)
        for m in rule.code_markers:
            matched_markers.append(m)

        code_evidence: List[Evidence] = []
        for cm in rule.code_markers:
            code_evidence.extend(find_marker_evidence(repo_root, cm, max_hits=max(1, max_evidence // 2), ignore_tests=ignore_tests))
            if len(code_evidence) >= max_evidence:
                break

        import_evidence: List[Evidence] = []
        for im in rule.import_markers:
            if matched_by_import:
                import_evidence.extend(find_marker_evidence(repo_root, im, max_hits=max(1, max_evidence // 2), ignore_tests=ignore_tests))
            if len(import_evidence) >= max_evidence:
                break

        go_mod_evidence: List[Evidence] = []
        if matched_by_mod:
            for im in rule.import_markers:
                go_mod_evidence.extend(
                    find_marker_evidence_in_files(
                        repo_root=repo_root,
                        relative_files=[f for f in go_mod_files if isinstance(f, str)],
                        marker=im,
                        max_hits=max(1, max_evidence // 2),
                        why="go.mod 依赖声明命中该组件/库",
                    )
                )
                if len(go_mod_evidence) >= max_evidence:
                    break

        if not (matched_by_mod or matched_by_import or code_evidence):
            continue

        confidence = 0.2
        reasons: List[str] = []
        if matched_by_mod:
            confidence += 0.25
            reasons.append("go.mod 依赖中命中该组件/库")
        if matched_by_import:
            confidence += 0.25
            reasons.append("Go 源码 import 命中该组件/库")
        if code_evidence:
            confidence += 0.35
            reasons.append("Go 源码中出现典型初始化/调用点")

        confidence = min(0.95, max(0.0, confidence))

        evidence: List[Evidence] = []
        evidence.extend(code_evidence[:max_evidence])
        if len(evidence) < max_evidence:
            evidence.extend(go_mod_evidence[: max_evidence - len(evidence)])
        if len(evidence) < max_evidence:
            evidence.extend(import_evidence[: max_evidence - len(evidence)])

        stacks.append(
            {
                "category": rule.category,
                "name": rule.name,
                "confidence": round(confidence, 2),
                "reasons": reasons,
                "evidence": [e.__dict__ for e in evidence[:max_evidence]],
            }
        )

        if verbose:
            print(f"识别到技术栈：{rule.category}/{rule.name}，置信度={confidence:.2f}", file=os.sys.stderr)

    stacks.sort(key=lambda x: (-x["confidence"], x["category"], x["name"]))
    return stacks


def _collect_deploy_related_files(repo_root: Path) -> Dict[str, List[str]]:
    """收集部署/运行相关文件（Docker/Compose/K8s/Helm）。"""

    dockerfiles: List[str] = []
    composefiles: List[str] = []
    charts: List[str] = []
    k8s_manifests: List[str] = []

    compose_names = {
        "docker-compose.yml",
        "docker-compose.yaml",
        "compose.yml",
        "compose.yaml",
    }

    for p in _walk_all_files(repo_root):
        if not p.is_file():
            continue
        rel = str(p.relative_to(repo_root))
        name = p.name
        lower = name.lower()

        if lower.startswith("dockerfile"):
            dockerfiles.append(rel)
            continue

        if name in compose_names:
            composefiles.append(rel)
            continue

        if name == "Chart.yaml":
            charts.append(rel)
            continue

        if name.endswith((".yaml", ".yml")):
            rel_lower = rel.lower()
            if any(seg in rel_lower for seg in ("/k8s/", "/kubernetes/", "/manifests/", "/deploy/", "/deployment/", "/helm/", "/charts/")):
                text = _read_text(p)
                if "apiVersion:" in text and "kind:" in text:
                    k8s_manifests.append(rel)
                    continue
            else:
                text = _read_text(p)
                if "apiVersion:" in text and "kind:" in text and "metadata:" in text:
                    k8s_manifests.append(rel)
                    continue

    dockerfiles.sort()
    composefiles.sort()
    charts.sort()
    k8s_manifests.sort()

    return {
        "dockerfiles": dockerfiles,
        "composefiles": composefiles,
        "charts": charts,
        "k8s_manifests": k8s_manifests,
    }


def detect_deploy_runtime(repo_root: Path, max_evidence: int) -> List[Dict]:
    """识别容器化/部署相关技术栈（Docker/Compose/K8s/Helm）。"""

    files = _collect_deploy_related_files(repo_root)
    stacks: List[Dict] = []

    dockerfiles = files.get("dockerfiles", [])
    if dockerfiles:
        markers = ("FROM ", "EXPOSE", "CMD", "ENTRYPOINT", "COPY --from=")
        evidence: List[Evidence] = []
        for m in markers:
            evidence.extend(
                find_marker_evidence_in_files(
                    repo_root=repo_root,
                    relative_files=dockerfiles,
                    marker=m,
                    max_hits=max(1, max_evidence // 2),
                    why="Dockerfile 中包含镜像构建/运行声明",
                )
            )
            if len(evidence) >= max_evidence:
                break

        confidence = 0.8
        if any("COPY --from=" in e.match for e in evidence):
            confidence = 0.9

        stacks.append(
            {
                "category": "deploy_runtime",
                "name": "docker",
                "confidence": round(confidence, 2),
                "reasons": [
                    "仓库中存在 Dockerfile（容器镜像构建与运行配置）",
                ],
                "evidence": [e.__dict__ for e in evidence[:max_evidence]],
            }
        )

    composefiles = files.get("composefiles", [])
    if composefiles:
        evidence = find_marker_evidence_in_files(
            repo_root=repo_root,
            relative_files=composefiles,
            marker="services:",
            max_hits=max_evidence,
            why="Compose 文件中定义了 services",
        )
        stacks.append(
            {
                "category": "deploy_runtime",
                "name": "docker-compose",
                "confidence": 0.85,
                "reasons": ["仓库中存在 docker-compose/compose 配置文件"],
                "evidence": [e.__dict__ for e in evidence[:max_evidence]],
            }
        )

    charts = files.get("charts", [])
    if charts:
        evidence = find_marker_evidence_in_files(
            repo_root=repo_root,
            relative_files=charts,
            marker="name:",
            max_hits=max_evidence,
            why="Helm Chart.yaml 中声明 chart 名称",
        )
        stacks.append(
            {
                "category": "deploy_runtime",
                "name": "helm",
                "confidence": 0.8,
                "reasons": ["仓库中存在 Helm Chart.yaml"],
                "evidence": [e.__dict__ for e in evidence[:max_evidence]],
            }
        )

    k8s = files.get("k8s_manifests", [])
    if k8s:
        evidence: List[Evidence] = []
        for m in ("apiVersion:", "kind:", "metadata:"):
            evidence.extend(
                find_marker_evidence_in_files(
                    repo_root=repo_root,
                    relative_files=k8s,
                    marker=m,
                    max_hits=max(1, max_evidence // 2),
                    why="Kubernetes 清单中存在关键字段",
                )
            )
            if len(evidence) >= max_evidence:
                break
        stacks.append(
            {
                "category": "deploy_runtime",
                "name": "kubernetes",
                "confidence": 0.8,
                "reasons": ["仓库中存在 Kubernetes manifests（yaml/yml）"],
                "evidence": [e.__dict__ for e in evidence[:max_evidence]],
            }
        )

    return stacks


def main(argv: Optional[Sequence[str]] = None) -> int:
    """命令行入口：输出 JSON 结果。"""

    parser = argparse.ArgumentParser(description="Go 后端技术栈初步扫描（输出 JSON）")
    parser.add_argument("repo_root", help="Go 仓库根目录（绝对路径或相对路径）")
    parser.add_argument("--max-evidence", type=int, default=3, help="每个技术栈最多输出多少条证据（默认 3）")
    parser.add_argument("--include-tests", action="store_true", help="是否包含测试代码（默认不包含）")
    parser.add_argument("--verbose", action="store_true", help="输出中文调试日志到 stderr")
    args = parser.parse_args(argv)

    repo_root = Path(args.repo_root).expanduser().resolve()
    ignore_tests = not args.include_tests

    if not repo_root.exists() or not repo_root.is_dir():
        print(json.dumps({"error": "repo_root 不存在或不是目录", "repo_root": str(repo_root)}, ensure_ascii=False))
        return 2

    go_mods = collect_go_mods(repo_root)
    if args.verbose:
        print(f"已发现 go.mod 数量：{len(go_mods)}", file=os.sys.stderr)

    stacks = detect_stacks(
        repo_root=repo_root,
        go_mods=go_mods,
        max_evidence=max(1, args.max_evidence),
        ignore_tests=ignore_tests,
        verbose=args.verbose,
    )
    stacks.extend(detect_deploy_runtime(repo_root=repo_root, max_evidence=max(1, args.max_evidence)))
    stacks.sort(key=lambda x: (-x.get("confidence", 0.0), x.get("category", ""), x.get("name", "")))

    output = {
        "repo": {
            "root": str(repo_root),
            "go_mods": go_mods,
        },
        "stacks": stacks,
        "notes": [
            "该脚本为启发式初筛：建议对高价值结论再补充入口/注册路径证据",
            "若仅在 go.mod 出现但未发现调用点，置信度通常不会很高",
        ],
    }
    print(json.dumps(output, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
