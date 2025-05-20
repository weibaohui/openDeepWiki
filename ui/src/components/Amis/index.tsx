import { registerRenderer, render as renderAmis, Schema} from 'amis'
import {AlertComponent, ToastComponent} from 'amis-ui'
import axios from 'axios'
import {fetcher} from "@/components/Amis/fetcher";
import WebSocketMarkdownViewerComponent from "@/components/Amis/custom/WebSocketMarkdownViewer.tsx";
import WebSocketViewerComponent from "@/components/Amis/custom/WebSocketViewer.tsx";
import WebSocketChatGPT from "@/components/Amis/custom/WebSocketChatGPT.tsx";
import GlobalTextSelector from '@/layout/TextSelectionPopover';
import PasswordEditorWithForm from "@/components/Amis/custom/PasswordEditorWithForm/PasswordEditorWithForm.tsx";
import SSELogDisplayComponent from "@/components/Amis/custom/LogView/SSELogDisplay.tsx";
// @ts-ignore
registerRenderer({type: 'webSocketMarkdownViewer', component: WebSocketMarkdownViewerComponent})
// @ts-ignore
registerRenderer({type: 'websocketViewer', component: WebSocketViewerComponent})
// @ts-ignore
registerRenderer({type: 'chatgpt', component: WebSocketChatGPT})

//@ts-ignore
registerRenderer({type: 'passwordEditor', component: PasswordEditorWithForm})
//@ts-ignore
registerRenderer({type: 'sseLogDisplay', component: SSELogDisplayComponent})

interface Props {
    schema: Schema
}


const Amis = ({schema}: Props) => {
    const theme = 'cxd';
    const locale = 'zh-CN';


    return <>
        <GlobalTextSelector/>

        <ToastComponent
            theme={theme}

            position={'top-center'}
            locale={locale}
        />
        <AlertComponent theme={theme} key="alert" locale={locale}/>
        {

            renderAmis(schema,
                {},
                {
                    theme: 'cxd',
                    updateLocation: () => {
                    },
                    fetcher,
                    isCancel: value => axios.isCancel(value),
                })
        }
    </>
}
export default Amis
