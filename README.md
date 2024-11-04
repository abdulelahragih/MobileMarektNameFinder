# Device Name Finder

This microservice helps you find the user-friendly name of Android devices based on the device's model and brand name.

## Setup

1. **Clone the repository:**
   ```bash
   git clone <repository_url>
   cd <repository_directory>
   ```

2. **Run the service with Docker:**
   Use Docker Compose to build and run the microservice.
   ```bash
   docker-compose up --build
   ```

   This command will start the service on `localhost:8089`.

## Database Initialization

The microservice fetches devices data from an external CSV file and populates it into an SQLite database. This file is accessed automatically during startup or by sending an update request.

## Endpoints

### 1. **Get Device Name**

   Retrieve the user-friendly name of a device by providing the brand and model.

   - **URL**: `http://localhost:8089/get-device-name`
   - **Method**: `POST`
   - **Payload**:
     ```json
     {
       "retail_branding": "Samsung",
       "model": "SM-G991B"
     }
     ```
   - **Response**:
     - Success (200):
       ```json
       {
         "data": "Galaxy S21 5G"
       }
       ```
     - Error (404): If the device is not found.
       ```json
       {
         "error": "Device not found"
       }
       ```

### 2. **Update Device Data**

   Fetches the latest device data from a public CSV file and updates the database. New records are added if they do not already exist.

   - **URL**: `http://localhost:8089/update-devices`
   - **Method**: `POST`
   - **Response**:
     - Success (200):
       ```
       Devices updated successfully
       ```
     - Error (500): If there is an issue updating the database.