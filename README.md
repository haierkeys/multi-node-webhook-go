# Golang Multi-node Webhook Tools
---

基于 qiniu/webhook 修改的 多节点webhook 更新工具

它能够做什么？简单来说，它就是一个让 Github/Bitbucket repo 在某个分支发生 push 行为的时候，自动触发一段脚本。

# 配置
vim xxx.conf
```json
{
    "bind": ":9876",
    "items": [
    {
        "repo": "https://github.com/qiniu/docs.qiniu.com",
        "branch": "master",
        "script": "update-qiniu-docs.sh"
    },
    {
        "repo": "https://bitbucket.org/Wuvist/angelbot/",
        "branch": "master",
        "script": "restart-angelbot.sh"
    }
]}
```

# 启动方法

- 源码启动:
```
go run webhook.go xxx.conf
```

- 执行程序启动
```
webhook xxx.conf
```

这样就启动 webhook 服务器了。


