import json
import csv
import sys

def json_to_csv(input_file, output_file):
    # 打开输入和输出文件
    with open(input_file, 'r', encoding='utf-8') as infile, \
         open(output_file, 'w', newline='', encoding='utf-8') as outfile:

        # 初始化CSV写入器
        writer = None

        # 逐行处理JSON
        for line in infile:
            # 解析JSON
            data = json.loads(line.strip())

            # 如果是第一行，创建CSV写入器并写入表头
            if writer is None:
                fieldnames = data.keys()
                writer = csv.DictWriter(outfile, fieldnames=fieldnames)
                writer.writeheader()

            # 写入数据行
            writer.writerow(data)

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python t.py <input_file> <output_file>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]
    json_to_csv(input_file, output_file)