#!/usr/bin/env python3
import subprocess
import csv
import itertools
from datetime import datetime
import argparse
import plotly.express as px
import pandas as pd

# 配置测试组（在此修改测试参数）
TEST_GROUPS = [
    {
        'type': 'string',
        'key_counts': [1000, 5000],
        'field_counts': [0],
        'field_sizes': [0],
        'key_sizes': [16, 64],
        'value_sizes': [64, 256]
    },
    {
        'type': 'hash',
        'key_counts': [500],
        'key_sizes': [32],
        'field_counts': [10, 50],
        'field_sizes': [16],
        'value_sizes': [128]
    }
]

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('-a', '--addr', default='localhost:6379', help='Redis 服务器地址')
    parser.add_argument('-p', '--password', default='', help='Redis 认证密码')
    return parser.parse_args()

def parse_memory(arr):
    memory_keys = {
        'used_memory:': 'used_memory',
        'total_system_memory:': 'total_system_memory',
        'maxmemory:': 'maxmemory'
    }
    
    result = {}
    
    for i, item in enumerate(arr):
        if item in memory_keys and i+1 < len(arr):
            key = memory_keys[item]
            value = arr[i+1].rstrip(',')
            try:
                result[key] = int(value)
            except ValueError:
                result[key] = value
                
    return result

def run_command(params, global_args):
    """执行 resp-cost 命令并返回内存指标"""
    args = ['./resp-cost', 'populate', '-e',
            '-a', global_args.addr,
            '-p', global_args.password]
    for k, v in params.items():
        if v is not None:
            args.append(f'--{k.replace("_", "-")}={v}')

    res = dict(params)
    try:
        # 执行测试命令
        result = subprocess.run(args, check=True, capture_output=True, text=True)
        res.update(parse_memory(result.stderr.strip().split()))
        return res
    except subprocess.CalledProcessError as e:
        print(f"命令执行失败: {e}")
        return None

def show_scatter(result):
    # 示例数据结构
    df = pd.DataFrame(result)

    fig = px.scatter_3d(df,
                    x='key_count',
                    y='field_count',
                    z='value_size',
                    size='used_memory',
                    hover_data=['type', 'field_count', 'key_size', 'field_count', 'field_size', 'value_size', 'used_memory'],  # 悬停显示字段数量
                    title="hash数据分析")

    fig.update_layout(scene=dict(
        xaxis_title='键数量',
        yaxis_title='键个数',
        zaxis_title='值个数'))
    fig.show()

def main():
    global_args = parse_args()

    for group in TEST_GROUPS:
        # 生成参数矩阵
        base_params = {
            'type': group['type'],
            'field_count': None,
            'field_size': None
        }
        
        # 构建参数组合
        params_list = []
        keys = ['key_count', 'key_size', 'field_count', 'field_size', 'value_size']
        combinations = itertools.product(
            group.get('key_counts', [1000]),
            group.get('key_sizes', [16]),
            group.get('field_counts', [0]),
            group.get('field_sizes', [0]),
            group.get('value_sizes', [64])
        )

        dataSet = []

        for combo in combinations:
            params = base_params.copy()
            for i, key in enumerate(keys):
                params[key] = combo[i]
            
            # 执行测试
            result = run_command({k:v for k,v in params.items() if v is not None}, global_args)
            dataSet.append(result)
        
        show_scatter(dataSet)

if __name__ == "__main__":
    main()