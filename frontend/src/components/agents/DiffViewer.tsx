import React from 'react';
import { Modal } from 'antd';

export interface DiffViewerProps {
  oldContent: string;
  newContent: string;
  open: boolean;
  onClose: () => void;
  fileName?: string;
  oldVersion?: number;
  newVersion?: number;
}

// 简单的行级差异计算
function computeDiff(oldLines: string[], newLines: string[]) {
  const result: Array<{ type: 'same' | 'added' | 'removed'; line: string; oldLineNum?: number; newLineNum?: number }> = [];
  let i = 0;
  let j = 0;

  while (i < oldLines.length || j < newLines.length) {
    if (i < oldLines.length && j < newLines.length) {
      if (oldLines[i] === newLines[j]) {
        result.push({ type: 'same', line: oldLines[i], oldLineNum: i + 1, newLineNum: j + 1 });
        i++;
        j++;
      } else {
        // 查找匹配行
        let found = false;
        // 向后查找在旧文本中找到新文本的当前行
        for (let k = 1; k <= 10 && j + k < newLines.length; k++) {
          if (oldLines[i] === newLines[j + k]) {
            // 将新文本中的 k 行标记为添加
            for (let m = 0; m < k; m++) {
              result.push({ type: 'added', line: newLines[j + m], newLineNum: j + m + 1 });
            }
            j += k;
            found = true;
            break;
          }
        }
        if (!found) {
          // 向后查找在新文本中找到旧文本的当前行
          for (let k = 1; k <= 10 && i + k < oldLines.length; k++) {
            if (oldLines[i + k] === newLines[j]) {
              // 将旧文本中的 k 行标记为删除
              for (let m = 0; m < k; m++) {
                result.push({ type: 'removed', line: oldLines[i + m], oldLineNum: i + m + 1 });
              }
              i += k;
              found = true;
              break;
            }
          }
        }
        if (!found) {
          result.push({ type: 'removed', line: oldLines[i], oldLineNum: i + 1 });
          result.push({ type: 'added', line: newLines[j], newLineNum: j + 1 });
          i++;
          j++;
        }
      }
    } else if (i < oldLines.length) {
      result.push({ type: 'removed', line: oldLines[i], oldLineNum: i + 1 });
      i++;
    } else {
      result.push({ type: 'added', line: newLines[j], newLineNum: j + 1 });
      j++;
    }
  }

  return result;
}

const DiffViewer: React.FC<DiffViewerProps> = ({
  oldContent,
  newContent,
  open,
  onClose,
  fileName,
  oldVersion,
  newVersion,
}) => {
  const oldLines = oldContent.split('\n');
  const newLines = newContent.split('\n');
  const diff = computeDiff(oldLines, newLines);

  const title = fileName && oldVersion !== undefined && newVersion !== undefined
    ? `${fileName} - 版本 ${oldVersion} vs 当前版本 (#${newVersion})`
    : '版本差异';

  return (
    <Modal
      title={title}
      open={open}
      onCancel={onClose}
      width={900}
      footer={null}
      style={{ top: 20 }}
    >
      <div style={{ maxHeight: '70vh', overflow: 'auto' }}>
        <div
          style={{
            fontFamily: 'Monaco, Consolas, monospace',
            fontSize: '13px',
            lineHeight: '1.5',
            backgroundColor: '#1e1e1e',
            color: '#d4d4d4',
            padding: '16px',
            borderRadius: '4px',
          }}
        >
          {diff.map((item, index) => {
            let bgColor = 'transparent';
            let color = '#d4d4d4';
            let prefix = '';

            if (item.type === 'removed') {
              bgColor = 'rgba(255, 100, 100, 0.2)';
              color = '#f48771';
              prefix = '-';
            } else if (item.type === 'added') {
              bgColor = 'rgba(100, 255, 100, 0.2)';
              color = '#73bf69';
              prefix = '+';
            }

            return (
              <div
                key={index}
                style={{
                  backgroundColor: bgColor,
                  padding: '2px 0',
                  display: 'flex',
                }}
              >
                <span
                  style={{
                    color: '#858585',
                    minWidth: '50px',
                    textAlign: 'right',
                    marginRight: '16px',
                    userSelect: 'none',
                  }}
                >
                  {item.type === 'removed'
                    ? item.oldLineNum
                    : item.type === 'added'
                    ? item.newLineNum
                    : item.oldLineNum}
                </span>
                {item.type !== 'same' && (
                  <span
                    style={{
                      color: item.type === 'removed' ? '#f48771' : '#73bf69',
                      marginRight: '8px',
                      userSelect: 'none',
                    }}
                  >
                    {prefix}
                  </span>
                )}
                <span style={{ color: color }}>{item.line || '\u00A0'}</span>
              </div>
            );
          })}
        </div>
      </div>
      <div style={{ marginTop: '16px' }}>
        <div style={{ display: 'flex', gap: '24px', fontSize: '13px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span
              style={{
                display: 'inline-block',
                width: '16px',
                height: '16px',
                backgroundColor: 'rgba(255, 100, 100, 0.2)',
                border: '1px solid #f48771',
              }}
            />
            <span style={{ color: '#d4d4d4' }}>删除行（历史版本中有，当前版本中没有）</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span
              style={{
                display: 'inline-block',
                width: '16px',
                height: '16px',
                backgroundColor: 'rgba(100, 255, 100, 0.2)',
                border: '1px solid #73bf69',
              }}
            />
            <span style={{ color: '#d4d4d4' }}>新增行（当前版本中有，历史版本中没有）</span>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default DiffViewer;
