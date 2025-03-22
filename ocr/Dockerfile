FROM python:3.12-slim

# Install tesseract + basic languages
RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-tha \
    libgl1 \
    poppler-utils \
    && rm -rf /var/lib/apt/lists/*

# Copy traineddata (คุณโหลดมาใน project folder)
COPY ./tha.traineddata /usr/share/tesseract-ocr/4.00/tessdata/tha.traineddata
COPY ./eng.traineddata /usr/share/tesseract-ocr/4.00/tessdata/eng.traineddata

# Set working directory
WORKDIR /app
COPY . /app

# Install Python packages
RUN pip install --no-cache-dir -r requirements.txt

# Set tessdata path
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/4.00/tessdata/

# Run FastAPI app
CMD ["uvicorn", "main_ocr:app", "--host", "0.0.0.0", "--port", "8000"]
