FROM python:3.13
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
# 실행 시 스케줄러가 무한 반복됨
CMD ["python", "fetch_notion_data.py"]