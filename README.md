Golang Multi-node Webhook Tools
===
>multi-node-webhook 是 golang编写的多节点远程脚本/命令执行工具

用于当请求通知到webhook来执行固定的脚本

### 使用场景示例 (CI/CD流程最后的更新代码/程序到服务器):

CI/CD 打包docker镜像后,推送到`阿里云`镜像仓库\
通过设置好的`触发器`通知`multi-node-webhook`,\
`multi-node-webhook`在接受到HTTP请求之后会解析POST参数,\
传递参数传递到各个节点服务器,各个节点服务器执行预设的`shell`脚本或`bash`命令



一. 启动 Run
---

* 启动参数
  | 参数  | 参数说明      | 必选    |
  |:-----|:-------------|:-------|
  | -c  | 配置文件路径 [缩写] | 是  |
  | -config | 配置文件路径 | 是,<br/> 和`-c`参数二选一    |
  | -host | 服务启动的绑定的host; <br/> 例如 "`:8080`" 则表示绑定并监听`8080`端口 | 否,<br/> 不提供会使用配置文件内的值 |
  | -id | 当前服务器的唯一ID | 是 |


- 源码启动:
    ```
    go run webhook.go -id master -c config.json
    ```

- 执行程序启动 (目前支持linux 和 windows)
    ```
    multi-node-webhook -id master -c config.json
    ```
  这样就启动 multi-node-webhook 服务器了。


二.配置 Config
---
- 完整的配置文件示例
  vim config.json
  ```json
  {
    "nodes" :[
      {
        "id": "master",
        "host": ":8888"
      },
      {
        "id": "node1",
        "host": "192.168.16.1:8888"
      }
    ],
    "hooks": [
      {
        "key": "update",
        "commands": [
          {
            "id": "master",
            "command": "ls",
            "display": true,
            "parm_bind": "{\"-a=\":\"\\\"tag\\\":\\\"([a-z0-9.-]+)\\\"\"}"
          },
          {
            "id": "node1",
            "command": "docker_redeploy.sh",
            "display": true,
            "parm_bind": "{\"var\":[\"push_data\",\"tag\"]}"
          }
        ]
      }
    ]
  }
  ```
- 配置参数说明
  - `nodes`为节点服务器列表
  - `hooks`为脚本列表
    - `id`对应的节点服务器
    - `command`需要执行的`脚本`或`命令`<br>***注意当 hook内没有定义command值则不会在对应的节点服务器执行,用于定义webhook中心服务器***
    - `display` 为是否打印运行信息
    - `parm_bind` 则用于设置将请求的POST内容通过正则匹配传递给`command`,
      - 正则的规则为 JSON(参数完整格式:正则表达式)<br>
      例如:```"{"-a=":"\"tag\"\:\"([a-z0-9.-]+)\""}```<br>
      匹配`POST`内容`"tag":"tag1"`中的`tag1`并将最终拼接为执行命令`command -a=tag1`
     - *注意当 `hook`内没有定义`command`值则不会在对应的节点服务器执行,用于定义`webhook中心服务器`* 

