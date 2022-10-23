FROM python:3.10

ENV KNOU_ID "id"
ENV KNOU_PW "pw"
ENV TELEGRAM_CHAT_ID "chat_id"
ENV TELEGRAM_API_TOKEN "token"
ENV DRIVER_COMMAND_URL "http://localhost:4444"

WORKDIR /usr/src

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

CMD [ "python", "main.py" ]
