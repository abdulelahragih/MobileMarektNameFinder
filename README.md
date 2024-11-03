# Device Name Finder

This microservice helps you find the user-friendly name of a device based on the device's model and brand name. It supports case-insensitive search and ensures that missing or empty fields are handled gracefully.

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

The microservice fetches device data from an external CSV file and populates it into an SQLite database. This file is accessed automatically during startup or by sending an update request.

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

## How It Works

1. **Database**: The service uses SQLite to store device information.
2. **Case-Insensitive Search**: The `/get-device-name` endpoint performs a case-insensitive search to retrieve the marketing name of a device.
3. **Missing Field Handling**: If any fields in the CSV file are empty, they are filled with a default value (`"Unknown"` or empty string) to ensure data consistency.

## Example Usage

After starting the service with Docker Compose, you can test it with `curl` or any HTTP client:

```bash
# Example: Get Device Name
curl -X POST http://localhost:8089/get-device-name \
-H "Content-Type: application/json" \
-d '{
      "retail_branding": "Samsung",
      "model": "SM-G991B"
    }'

# Example: Update Devices Database
curl -X POST http://localhost:8089/update-devices
```

## Notes

- Ensure Docker is installed and running before executing `docker-compose up`.
- The service is configured to run on port `8089` and can be accessed at `http://localhost:8089`.