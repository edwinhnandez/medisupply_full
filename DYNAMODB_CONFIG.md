# DynamoDB Configuration Guide

This document explains how to configure the system to use external DynamoDB instead of the local DynamoDB Local instance.

## Current Setup (Local DynamoDB)

The system currently uses DynamoDB Local for development/testing with dummy credentials.

## Switching to External DynamoDB

### 1. Environment Variables to Update

For each microservice, update the following environment variables in their respective `k8s-deployment.yaml` files:

#### Python Service (movimiento-inventario)
```yaml
env:
- name: DYNAMODB_ENDPOINT_URL
  value: ""  # Remove this line or set to empty for AWS DynamoDB
- name: DYNAMODB_REGION
  value: "us-east-1"  # Change to your AWS region
- name: DYNAMODB_ACCESS_KEY
  value: "YOUR_AWS_ACCESS_KEY_ID"  # Replace with real AWS access key
- name: DYNAMODB_SECRET_KEY
  value: "YOUR_AWS_SECRET_ACCESS_KEY"  # Replace with real AWS secret key
```

#### Go Services (orden-compra, proveedor)
```yaml
env:
- name: AWS_REGION
  value: "us-east-1"  # Change to your AWS region
- name: AWS_ACCESS_KEY_ID
  value: "YOUR_AWS_ACCESS_KEY_ID"  # Replace with real AWS access key
- name: AWS_SECRET_ACCESS_KEY
  value: "YOUR_AWS_SECRET_ACCESS_KEY"  # Replace with real AWS secret key
```

### 2. Required DynamoDB Tables

The following tables need to be created in your AWS DynamoDB:

#### For movimiento-inventario service:
- `movimiento-inventario-events`
- `movimiento-inventario-read`

#### For orden-compra service:
- `orden-compra-events`
- `orden-compra-read`

#### For proveedor service:
- `proveedor-events`
- `proveedor-read`

#### For ingreso-inventario service:
- `ingreso-inventario-events`
- `ingreso-inventario-read`

### 3. Table Schema

Each table should have the following structure:

#### Events Tables (for CQRS write model):
```json
{
  "TableName": "movimiento-inventario-events",
  "KeySchema": [
    {
      "AttributeName": "id",
      "KeyType": "HASH"
    },
    {
      "AttributeName": "timestamp",
      "KeyType": "RANGE"
    }
  ],
  "AttributeDefinitions": [
    {
      "AttributeName": "id",
      "AttributeType": "S"
    },
    {
      "AttributeName": "timestamp",
      "AttributeType": "S"
    }
  ],
  "BillingMode": "PAY_PER_REQUEST"
}
```

#### Read Tables (for CQRS read model):
```json
{
  "TableName": "movimiento-inventario-read",
  "KeySchema": [
    {
      "AttributeName": "id",
      "KeyType": "HASH"
    }
  ],
  "AttributeDefinitions": [
    {
      "AttributeName": "id",
      "AttributeType": "S"
    }
  ],
  "BillingMode": "PAY_PER_REQUEST"
}
```

### 4. AWS IAM Permissions

Ensure your AWS credentials have the following DynamoDB permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:DescribeTable"
      ],
      "Resource": [
        "arn:aws:dynamodb:us-east-1:*:table/movimiento-inventario-*",
        "arn:aws:dynamodb:us-east-1:*:table/orden-compra-*",
        "arn:aws:dynamodb:us-east-1:*:table/proveedor-*",
        "arn:aws:dynamodb:us-east-1:*:table/ingreso-inventario-*"
      ]
    }
  ]
}
```

### 5. Deployment Steps

1. **Create Tables**: Use AWS CLI or Console to create the required tables
2. **Update Environment Variables**: Modify the deployment files as shown above
3. **Deploy**: Apply the updated configurations
4. **Test**: Verify the services can connect to external DynamoDB

### 6. Security Best Practices

- Use IAM roles instead of access keys when possible
- Implement least privilege access
- Use AWS Secrets Manager for sensitive credentials
- Enable DynamoDB encryption at rest
- Use VPC endpoints for private network access

### 7. Monitoring

- Enable CloudWatch metrics for DynamoDB
- Set up alarms for throttling and errors
- Monitor costs and usage patterns
- Use AWS X-Ray for distributed tracing

## Quick Switch Script

A script can be created to quickly switch between local and external DynamoDB configurations.
