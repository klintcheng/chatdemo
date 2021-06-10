import { w3cwebsocket, IMessageEvent, ICloseEvent } from 'websocket';
import { Buffer } from 'buffer';

export const Ping = new Uint8Array([0, 100, 0, 0, 0, 0])
export const Pong = new Uint8Array([0, 101, 0, 0, 0, 0])

export let sleep = async (second: number): Promise<void> => {
    return new Promise((resolve, _) => {
        setTimeout(() => {
            resolve()
        }, second * 1000)
    })
}

const StatusSuccess = "Success"
const StatusTimeout = "Timeout"
const StatusLoginfailed = "LoginFailed"

export let doLogin = async (url: string): Promise<{ status: string, conn: w3cwebsocket }> => {
    const LoginTimeout = 5 // 5 seconds
    return new Promise((resolve, reject) => {
        let conn = new w3cwebsocket(url)
        conn.binaryType = "arraybuffer"

        // 设置一个登陆超时器
        let tr = setTimeout(() => {
            resolve({ status: StatusTimeout, conn: conn });
        }, LoginTimeout * 1000);

        conn.onopen = () => {
            console.info("websocket open - readyState:", conn.readyState)

            if (conn.readyState === w3cwebsocket.OPEN) {
                clearTimeout(tr)
                resolve({ status: StatusSuccess, conn: conn });
            }
        }
        conn.onerror = (error: Error) => {
            clearTimeout(tr)
            console.error(error)
            resolve({ status: StatusLoginfailed, conn: conn });
        }
    })
}

const heartbeatInterval = 10 // seconds


export class IMClient {
    wsurl: string
    state: number
    private conn: w3cwebsocket | null
    private lastRead: number
    constructor(url: string, user: string) {
        this.wsurl = `${url}?user=${user}`
        this.state = w3cwebsocket.CLOSED
        this.conn = null
        this.lastRead = Date.now()

        this.heartbeatLoop()
        this.readDeadlineLoop()
    }
    async login(): Promise<{ status: string }> {
        if (this.state != w3cwebsocket.CLOSED) {
            return { status: "StateError" }
        }
        let { status, conn } = await doLogin(this.wsurl)
        console.info("login - ", status)

        if (status !== StatusSuccess) {
            return { status }
        }
        // overwrite onmessage
        conn.onmessage = (evt: IMessageEvent) => {
            try {
                this.lastRead = Date.now()

                let buf = Buffer.from(<ArrayBuffer>evt.data)
                let command = buf.readInt16BE(0)
                let len = buf.readInt32BE(2)
                console.info(`command:${command} len: ${len}`)
                if (command == 101) {
                    console.info("received a pong")
                }
            } catch (error) {
                console.error(evt.data, error)
            }
        }
        conn.onerror = (error) => {
            console.info("websocket error: ", error)
            this.errorHandler(error)
        }
        conn.onclose = (e: ICloseEvent) => {
            console.debug("connection onclose")
            this.errorHandler(new Error(e.reason))
        }
        this.conn = conn
        this.state = w3cwebsocket.OPEN
        return { status }
    }
    private heartbeatLoop() {
        console.debug("heartbeatLoop start")

        let loop = () => {
            if (this.state == w3cwebsocket.OPEN) {
                console.log("send ping")
                this.send(Ping)
            }
            setTimeout(loop, heartbeatInterval * 1000)
        }
        loop.call(this)
    }
    private readDeadlineLoop() {
        console.debug("deadlineLoop start")
        let loop = () => {
            if (this.state == w3cwebsocket.OPEN) {
                if ((Date.now() - this.lastRead) > 3 * heartbeatInterval * 1000) {
                    // read timeout
                    this.errorHandler(new Error("read timeout"))
                }
            }
            setTimeout(loop, 1000)
        }
        loop.call(this)
    }
    private async errorHandler(error: Error) {
        // 检查是否正常关闭
        if (this.state == w3cwebsocket.CLOSED || this.state == w3cwebsocket.CLOSING) {
            return
        }
        console.log(error)
        // 重连10次
        for (let index = 0; index < 10; index++) {
            try {
                console.info("try to relogin")
                let { status } = await this.login()
                if (status == "Success") {
                    return
                }
            } catch (error) {
                console.warn(error)
            }
            await sleep(3)
        }
        this.state == w3cwebsocket.CLOSED
        this.conn = null
        console.warn("程序中止")
    }
    private send(data: Buffer | Uint8Array): boolean {
        try {
            if (this.conn == null) {
                return false
            }
            this.conn.send(data)
        } catch (error) {
            // handle write error
            this.errorHandler(new Error("read timeout"))
            return false
        }
        return true
    }
}
