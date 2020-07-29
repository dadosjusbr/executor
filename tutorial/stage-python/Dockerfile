FROM python:3.7.2-slim

# Create output folder
RUN mkdir /output

# Move to working directory /app
WORKDIR /app

# Copy the code into the container
COPY . .

# Installing dependencies
RUN pip install --upgrade pip
RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "./script.py"]