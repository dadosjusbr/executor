FROM python:3.7.2-slim

# Move to working directory /app
WORKDIR /app

# Copy the code into the container
COPY . .

RUN pip install --upgrade pip

RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "./script.py"]