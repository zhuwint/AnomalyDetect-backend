import pandas as pd
import requests
import json
import argparse


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
                "threshold_upper": 90.0,
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


def create_stream_task(idx: int, sensor_mac: str):
    data = {
        "project_id": 3,
        "detect_model": {
            "name": "gumbel.real",
            "params": {
                "alpha": 0.000001,
                "freq": 48
            }
        },
        "target": {
            "sensor_mac": sensor_mac,
            "receive_no": "2",
            "sensor_type": "temperature_air"
        },
        "independent": [],
        # "model_update": {
        #     "interval": "24h",
        #     "query": {
        #         "measurement": "sensor_data",
        #         "range": {
        #             "start": "-720h",
        #             "stop": "now()"
        #         },
        #         "aggregate": "30m"
        #     }
        # },
        "anomaly_detect": {
            "duration": "0m"
        },
        "is_stream": True,
        "level": 1
    }
    req = requests.post(url="http://10.203.96.205:3030/api/task/stream", data=json.dumps(data).encode())
    if req.status_code < 200 or req.status_code > 299:
        print(f"创建失败：{sensor_mac} {req.content}")
    elif json.loads(req.text).get('status') == 0:
        print(f"创建成功")
    else:
        print(f"创建失败: {json.loads(req.text).get('msg')}")


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--kind', type=int, default=0)
    parser.add_argument('--all', type=int, default=0)
    arg = parser.parse_args()

    creater = None
    if arg.kind == 0:
        creater = create_stream_task
    else:
        creater = create_union_task
    sensor_macs = pd.read_csv('./sensor.csv')
    for i in range(len(sensor_macs)):
        creater(i, sensor_macs.iloc[i]["sensor_mac"])
        if arg.all == 0:
            break
