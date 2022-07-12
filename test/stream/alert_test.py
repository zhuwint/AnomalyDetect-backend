from tkinter.tix import Tree
from numpy import loadtxt
import pandas as pd 
from influxdb_client import InfluxDBClient
from pyrsistent import thaw

def load_ts(filePath: str) -> pd.DataFrame:
    data = pd.read_csv(filePath, index_col='time', usecols=['time', 'value'], encoding='utf16')
    data.index = pd.to_datetime(data.index)
    return data


def build_alert():
    t_l = 10.0
    t_h = 29.5
    data = load_ts("../2521_temperature.csv")
    alert = False 
    result = pd.DataFrame({'time': [], "alert": []}, index=None)
    for i in range(len(data)):
        value = data.iloc[i]['value']
        if t_l <= value <= t_h:
            if alert:
                result = result.append({"time": data.index[i], "alert": False}, ignore_index=True)
            alert = False
        else:
            if not alert:
                result = result.append({"time": data.index[i], "alert": True}, ignore_index=True)
            alert = True
    result.to_csv("./alert_result.csv", index=None)
    return result


def load_record():
    client = InfluxDBClient(url="http://10.203.96.205:8086",
                            token="Z-cI9B2RU_AScfo2YbF528VHdNtq7iAnhkg_HLr7itaGIfjKnXD4q_qTrUpiGauSaCmdBNAs0pMB8cyRLDn48A==",
                            org="my-org")
    queryApi = client.query_api()
    tables = queryApi.query('''
        from(bucket: "yinao")
        |> range(start: 2021-05-01T00:00:00Z, stop: 2022-01-01T12:00:00Z)
        |> filter(fn: (r) => r["_measurement"] == "alert_logs" and r["task_id"] == "aaaaailadplldbe" 
            and r["project_id"] == "3" 
            and r["sensor_mac"] == "C125" 
            and r["sensor_type"] == "temperature_air"
        )
        |> filter(fn: (r) => r["_field"] == "alert")'''
                            )
    data = pd.DataFrame({'time': [], "alert": []}, index=None)
    for table in tables:
        for row in table.records:
            # d = datetime.datetime.strptime(row.values["_time"], "%Y-%m-%dT%H%M%SZ")
            d = row.values["_time"]
            # t = d.timetuple()
            data = data.append({"time": d, "alert": True if row.values["_value"] == 1 else False}, ignore_index=True)
    data.time = pd.to_datetime(data.time)
    data.to_csv("./alert_result2.csv", index=None)
    return data


def compare(result1: pd.DataFrame, result2: pd.DataFrame):
    if len(result1) != len(result2):
        print(f"length {len(result1)} {len(result2)} not match")
        return False
    p = 0
    while p < len(result1):
        if (result1.index[p] != result2.index[p]) or (result1.iloc[p].alert != result2.iloc[p].alert):
            print(f"row {p+1} not match")
            print(result1.iloc[p])
            print(result2.iloc[p])
            return False
        p += 1
    return True

if __name__ == "__main__":
    df1 = build_alert()
    df2 = load_record()
    if compare(df1, df2):
        print("success")
    else:
        print("failed")
    