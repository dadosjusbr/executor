import sys
import pandas as pd
import json

data = sys.stdin.read()  
data = json.loads(data)
df = pd.json_normalize(data.get('employees'))
file_name = '{}-{}-{}-{}.csv'.format(data.get('aid'), data.get('month'), data.get('year'), data.get('timestamp'))
df.to_csv(file_name, index=False)
print("File {} was saved!".format(file_name))