import sys
import pandas as pd
import json
import os


data = sys.stdin.read()  
data = json.loads(data)
df = pd.json_normalize(data.get('employees'))

output = os.environ['OUTPUT_FOLDER']
file_name = '{}/{}-{}-{}-{}.csv'.format(output, data.get('aid'), data.get('month'), data.get('year'), data.get('timestamp'))

df.to_csv(file_name, index=False)
