告警引擎接收告警请求进行告警判断，并推送给用户。主要功能有：告警分级、周期告警、多途径推送等。
1.	告警分级：按照紧急程度从小到大将告警等级分为Info、Warnning、Alert三个等级。在周期告警时不同的紧急程度有不同的告警周期。紧急程度越高，告警周期就越小。
2.	周期告警：周期告警用于某一任务在告警状态解除前，按照一定周期重复推送告警消息。
3.	多途径推送：实现了基于电子邮件的推送以及基于Webhook的互联网应用推送（钉钉等）。

## API设计

### 获取用户订阅
API: /subscribe

Method: GET

Params: topic="" & userId=""

Data: application/json

```json
{
    "data": [
        {
            "projectId": "", // 项目ID
            "taskId": "",   // 任务ID
            "to": [
                {
                    "userId": "",    // 用户ID
                    "type": "",      // 接收器类型
                    "address": "",   // 接收器地址
                    "token": ""      // 认证信息
                },
                {
                    "userId": "",
                    "type": "",
                    "address": "",
                    "token": ""
                }
            ]
        }
    ]
}
```

### 创建用户订阅
API: /subscribe

Method: POST

Data: application/json
```json
{
    "projectId": "", // 项目ID
    "taskId": "",   // 任务ID
    "to": [
        {
            "userId": "",    // 用户ID
            "type": "",      // 接收器类型
            "address": "",   // 接收器地址
            "token": ""      // 认证信息
        },
        {
            "userId": "",
            "type": "",
            "address": "",
            "token": ""
        }
    ]
}
```

### 修改用户订阅
API: /subscribe

Method: PATCH

Params: topic="" & userId=""

Data: application/json
```json
{
    "to": [
        {
            "userId": "",
            "type": "",
            "address": "",
            "token": ""
        },
        {
            "userId": "",
            "type": "",
            "address": "",
            "token": ""
        }
    ]
}
```

### 删除用户订阅
API : /subscribe

Method: DELETE

Data: application/json


### 告警推送
API: /alert

Method: POST

Data: application/json

```json
{
    "projectId": 0,
    "taskId": "",
    "message": "",
    "level": 0
}
```