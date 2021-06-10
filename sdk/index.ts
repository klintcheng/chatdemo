import { IMClient } from "./jdk";

const main = async () => {
    let cli = new IMClient("ws://localhost:8000", "test_sdk");
    let { status } = await cli.login()
    console.log(status)
}

main()