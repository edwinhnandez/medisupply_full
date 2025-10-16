# DynamoDB Configuration Guide

This guide explains how to configure the Event Mesh system to use external DynamoDB instead of the local DynamoDB Local instance.

## Current Setup (Local DynamoDB)

The system is currently configured to use DynamoDB Local for development and testing. This setup uses dummy credentials and runs within the Kubernetes cluster.

## Switching to External DynamoDB (AWS)

### Quick Start

1. **Use the automated script** (recommended):
   ```bash
   ./scripts/switch-dynamodb.sh external --region us-east-1 --access-key YOUR_KEY --secret-key YOUR_SECRET
   ```

2. **Create DynamoDB tables**:
   ```bash
   ./scripts/setup-aws-dynamodb.sh --region us-east-1
   ```

3. **Restart services**:
   ```bash
   kubectl rollout restart deployment/movimiento-inventario -n event-mesh-system
   kubectl rollout restart deployment/orden-compra -n event-mesh-system
   kubectl rollout restart deployment/proveedor -n event-mesh-system
   ```

### Manual Configuration

If you prefer to configure manually, follow these steps:

#### 1. Update Python Service (movimiento-inventario)

Edit `movimiento-inventario/k8s-deployment.yaml`:

```yaml
env:
# Remove or comment out this line for external DynamoDB:
# - name: DYNAMODB_ENDPOINT_URL
#   value: "http://dynamodb-local:8000"

# Update these values:
- name: DYNAMODB_REGION
  value: "us-east-1"  # Change to your AWS region
- name: DYNAMODB_ACCESS_KEY
  value: "YOUR_AWS_ACCESS_KEY_ID"  # Replace with your AWS access key
- name: DYNAMODB_SECRET_KEY
  value: "YOUR_AWS_SECRET_ACCESS_KEY"  # Replace with your AWS secret key
```

#### 2. Update Go Services (orden-compra, proveedor)

Edit `orden-compra/k8s-deployment.yaml` and `proveedor/k8s-deployment.yaml`:

```yaml
env:
# Remove or comment out this line for external DynamoDB:
# - name: DYNAMODB_ENDPOINT
#   value: "http://dynamodb-local:8000"

# Update these values:
- name: DYNAMODB_REGION
  value: "us-east-1"  # Change to your AWS region
- name: AWS_ACCESS_KEY_ID
  value: "YOUR_AWS_ACCESS_KEY_ID"  # Replace with your AWS access key
- name: AWS_SECRET_ACCESS_KEY
  value: "YOUR_AWS_SECRET_ACCESS_KEY"  # Replace with your AWS secret key
```

### Required DynamoDB Tables

The following tables must be created in your AWS DynamoDB:

#### Table List
- `movimiento-inventario-events`
- `movimiento-inventario-read`
- `orden-compra-events`
- `orden-compra-read`
- `proveedor-events`
- `proveedor-read`
- `ingreso-inventario-events`
- `ingreso-inventario-read`

#### Table Schema

**Events Tables** (for CQRS write model):
```json
{
  "TableName": "movimiento-inventario-events",
  "KeySchema": [
    {"AttributeName": "id", "KeyType": "HASH"},
    {"AttributeName": "timestamp", "KeyType": "RANGE"}
  ],
  "AttributeDefinitions": [
    {"AttributeName": "id", "AttributeType": "S"},
    {"AttributeName": "timestamp", "AttributeType": "S"}
  ],
  "BillingMode": "PAY_PER_REQUEST"
}
```

**Read Tables** (for CQRS read model):
```json
{
  "TableName": "movimiento-inventario-read",
  "KeySchema": [
    {"AttributeName": "id", "KeyType": "HASH"}
  ],
  "AttributeDefinitions": [
    {"AttributeName": "id", "AttributeType": "S"}
  ],
  "BillingMode": "PAY_PER_REQUEST"
}
```

### AWS IAM Permissions

Your AWS credentials need the following DynamoDB permissions:

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
        "arn:aws:dynamodb:*:*:table/movimiento-inventario-*",
        "arn:aws:dynamodb:*:*:table/orden-compra-*",
        "arn:aws:dynamodb:*:*:table/proveedor-*",
        "arn:aws:dynamodb:*:*:table/ingreso-inventario-*"
      ]
    }
  ]
}
```

### Security Best Practices

1. **Use IAM Roles**: Instead of access keys, use IAM roles when possible
2. **AWS Secrets Manager**: Store credentials in AWS Secrets Manager
3. **Least Privilege**: Grant only the minimum required permissions
4. **Encryption**: Enable DynamoDB encryption at rest
5. **VPC Endpoints**: Use VPC endpoints for private network access
6. **Monitoring**: Enable CloudWatch metrics and alarms

### Switching Back to Local DynamoDB

To switch back to local DynamoDB:

```bash
./scripts/switch-dynamodb.sh local
```

### Troubleshooting

#### Common Issues

1. **Table Not Found**: Ensure all required tables exist in your AWS region
2. **Access Denied**: Check IAM permissions for your AWS credentials
3. **Wrong Region**: Verify the region matches where your tables are created
4. **Network Issues**: Check VPC and security group configurations

#### Verification Commands

```bash
# List tables in your region
aws dynamodb list-tables --region us-east-1

# Check table status
aws dynamodb describe-table --table-name movimiento-inventario-events --region us-east-1

# Test connection from a pod
kubectl run test-dynamodb --image=amazon/aws-cli:2.13.0 --restart=Never -n event-mesh-system -- aws dynamodb list-tables --region us-east-1
```

### Configuration Files

- `configs/dynamodb-external.yaml` - External DynamoDB configuration template
- `scripts/switch-dynamodb.sh` - Script to switch between local and external
- `scripts/setup-aws-dynamodb.sh` - Script to create DynamoDB tables in AWS
- `DYNAMODB_CONFIG.md` - Detailed configuration documentation

### Support

For issues or questions:
1. Check the logs: `kubectl logs -f deployment/movimiento-inventario -n event-mesh-system`
2. Verify table existence: `aws dynamodb list-tables --region YOUR_REGION`
3. Test connectivity: Use the verification commands above
