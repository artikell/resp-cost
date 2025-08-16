#!/usr/bin/env python3
import subprocess
import itertools
from datetime import datetime
import argparse
import json

TEST_GROUPS = json.load(open('./case.json'))

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('-a', '--addr', default='localhost:6379', help='服务器地址')
    parser.add_argument('-p', '--password', default='', help='认证密码')
    parser.add_argument('-t', '--target', default='', help='服务器类型, 如: redis, valkey')
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

    try:
        # 执行测试命令
        result = subprocess.run(args, check=True, capture_output=True, text=True)
        return parse_memory(result.stdout.strip().split())
    except subprocess.CalledProcessError as e:
        print(f"命令执行失败: {e}")
        return None

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
        keys = ['key_count', 'key_size', 'field_count', 'field_size', 'value_size']
        combinations = itertools.product(
            group.get('key_counts', [1000]),
            group.get('key_sizes', [16]),
            group.get('field_counts', [0]),
            group.get('field_sizes', [0]),
            group.get('value_sizes', [64])
        )

        for combo in combinations:
            params = base_params.copy()
            
            for i, key in enumerate(keys):
                params[key] = combo[i]
            
            # 执行测试
            result = run_command({k:v for k,v in params.items() if v is not None}, global_args)
            
            if result is None:
                continue

            params.update(result)
            params['target'] = global_args.target
            print(json.dumps(params), flush=True)

if __name__ == "__main__":
    main()