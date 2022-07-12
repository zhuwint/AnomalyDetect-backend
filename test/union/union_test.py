from time import time
import pandas as pd
import matplotlib.pyplot as plt
import requests
import argparse


def load_ts(filePath: str) -> pd.DataFrame:
    data = pd.read_csv(filePath, index_col='time', usecols=['time', 'value'], encoding='utf16')
    # data.index = pd.to_datetime(data.index).astype('int64')
    data.index = pd.to_datetime(data.index)
    # data = data["2021-07-29 06:30:00":"2021-07-29 07:30:00"]
    data.index = data.index.astype('int64')
    return data


def build_point(receiveNo: str, sensorMac: str, sensorType: str, value: float, timestamp: int) -> str:
    # sensor_data,project_id=2,receive_no=0,sensor_mac=829C,sensor_type=receive_time value=1645583400960.0 1645582800000000000
    return f'sensor_data,project_id=3,receive_no={receiveNo},sensor_mac={sensorMac},sensor_type={sensorType} value={value} {timestamp}'


def write(p: str):
    r = requests.post('http://localhost:3030/api/v2/write', data=p, params={'precision': 'ns'})
    if r.status_code < 200 or r.status_code > 299:
        print(f"写入失败: {r.content}")
    else:
        print("写入成功")


def write2(p: str):
    df = pd.DataFrame({'data': [p]})
    df.to_csv("./data.csv", mode='a', index=None, header=False)


def test_case(filename: str):
    file = open(filename)
    line = file.readline()
    while line:
        write(line)
        line = file.readline()
    file.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--case', type=int, default=1)
    arg = parser.parse_args()
    if arg.case == 0:
        test_case("./case_0.txt")
    elif arg.case == 1:
        test_case("./case_1.txt")
    elif arg.case == 2:
        test_case("./case_2.txt")
    elif arg.case == 3:
        test_case("./case_3.txt")
        