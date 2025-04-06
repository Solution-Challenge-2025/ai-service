#!/bin/bash

# Exit on error
set -e

# Check if project ID is provided
if [ -z "$1" ]; then
    echo "Please provide a project ID"
    echo "Usage: ./setup_gcloud.sh PROJECT_ID"
    exit 1
fi

PROJECT_ID=$1

# Set the project
echo "Setting Google Cloud project to: $PROJECT_ID"
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "Enabling required APIs..."
gcloud services enable vertexai.googleapis.com
gcloud services enable aiplatform.googleapis.com

# Create service account
echo "Creating service account..."
SA_NAME="analytics-ai-service"
SA_EMAIL="$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"

gcloud iam service-accounts create $SA_NAME \
    --display-name="Analytics AI Service Account" \
    --description="Service account for Analytics AI service"

# Grant necessary roles
echo "Granting necessary roles..."
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SA_EMAIL" \
    --role="roles/aiplatform.user"

# Create and download service account key
echo "Creating service account key..."
gcloud iam service-accounts keys create credentials.json \
    --iam-account=$SA_EMAIL

echo "Setup completed successfully!"
echo "The service account credentials have been saved to credentials.json"
echo "Please update your .env file with:"
echo "GOOGLE_CLOUD_PROJECT=$PROJECT_ID" 