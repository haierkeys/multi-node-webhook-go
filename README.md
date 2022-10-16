# Golang Multi-node Webhook Tools
---
>multi-node-webhook 是 golang编写的多节点脚本执行工具

多节点执行脚本和更新工具

用于当请求通知到webhook来执行固定的脚本

### 使用场景示例:
阿里云docker镜像仓库在接受镜像更新后,\
通过`触发器`通知`multi-node-webhook`,\
`multi-node-webhook`会解析参数,\
并执行这些参数的节点服务器`shell`脚本或`bash`命令




### Config 配置
vim config.json
```json
{
  "nodes" :[
    {
      "id": "master",
      "host": "192.168.16.1:8888"
    },
    {
      "id": "node1",
      "host": "192.168.16.1"
    }
  ],
  "hooks": [
    {
      "key": "update",
      "commands": [
        {
          "id": "master",
          "command": "restart-angelbot.sh"
        }
      ]
    }
  ]
}
```

### run 启动方法
    
- 源码启动:
    ```
    go run webhook.go config.json
    ```

- 执行程序启动
    ```
    webhook config.json
    ```
    这样就启动 webhook 服务器了。


