import pandas as pd
import requests
import json 


def create_union_task(idx: int, sensor_mac: str):
    data = {
        "task_name": "基准测试" + sensor_mac,
        "project_id": 3,
        "bucket": "yinao",
        "measurement": "sensor_data",
        "series": [
            {
                "sensor_mac": sensor_mac,
                "receive_no": "2",
                "sensor_type": "temperature_air",
                "threshold_upper": 29.5,
                "threshold_lower": 10.0
            },
            {
                "sensor_mac": sensor_mac,
                "receive_no": "3",
                "sensor_type": "humidity_air",
                "threshold_upper": 85.0,
                "threshold_lower": 45.0
            }
        ],
        "operate": [0],
        "duration": "10m",
        "is_stream": True,
        "level": 1
    }
    req = requests.post(url="http://10.203.96.205:3030/api/task/union", data=json.dumps(data).encode())
    if req.status_code < 200 or req.status_code > 299:
        print(f"创建失败: {req.content}")
    else:
        print("创建成功")


if __name__ == '__main__':
    sensor_macs = pd.read_csv('./sensor.csv')
    for i in range(len(sensor_macs)):
        create_union_task(i, sensor_macs.iloc[i]["sensor_mac"])
