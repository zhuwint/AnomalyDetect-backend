from time import time
import pandas as pd
import matplotlib.pyplot as plt
import requests


def load_ts(filePath: str) -> pd.DataFrame:
    data = pd.read_csv(filePath, index_col='time', usecols=['time', 'value'], encoding='utf16')
    data.index = pd.to_datetime(data.index).astype('int64')
    return data


def build_point(receiveNo: str, sensorMac: str, sensorType: str, value: float, timestamp: int) -> str:
    # sensor_data,project_id=2,receive_no=0,sensor_mac=829C,sensor_type=receive_time value=1645583400960.0 1645582800000000000
    return f'sensor_data,project_id=19,receive_no={receiveNo},sensor_mac={sensorMac},sensor_type={sensorType} value={value} {timestamp}'

def write(p: str):
    r = requests.post('http://localhost:3030/api/v2/write', data=p, params={'precision': 'ns'})
    if r.status_code < 200 or r.status_code > 299:
        print(f"写入失败: {r.content}")
    else:
        print("写入成功")


def test_union():
    humidity = load_ts('./2521_humidity.csv')
    temperature = load_ts('./2521_temperature.csv')
    p1, p2 = 0, 0
    count = 0
    while(p1 < len(humidity) and p1 < len(temperature)):
        point = ""
        if humidity.index[p1] < temperature.index[p2]:
            point = build_point("3", "2521", "humidity_air", humidity.iloc[p1].value, humidity.index[p1])
            p1 += 1
        else:
            point = build_point("2", "2521", "temperature_air", temperature.iloc[p1].value, temperature.index[p2])
            p2 += 1
        write(point)
        count += 1
        if count == 100:
            break


if __name__ == '__main__':
    print('start')
    test_union()
