#!/bin/bash
# Example dynamic inventory script
# This script generates inventory dynamically (e.g., from cloud provider API, database, etc.)
# It must output JSON format when called with --list

if [ "$1" = "--list" ]; then
    cat <<EOF
{
  "hosts": [
    {
      "name": "dynamic-web1",
      "address": "10.0.1.10",
      "port": 22,
      "user": "ubuntu",
      "vars": {
        "onigirazu_host": "10.0.1.10",
        "instance_type": "t3.medium",
        "cloud_provider": "aws",
        "region": "us-east-1"
      }
    },
    {
      "name": "dynamic-web2",
      "address": "10.0.1.11",
      "port": 22,
      "user": "ubuntu",
      "vars": {
        "onigirazu_host": "10.0.1.11",
        "instance_type": "t3.medium",
        "cloud_provider": "aws",
        "region": "us-east-1"
      }
    },
    {
      "name": "dynamic-db1",
      "address": "10.0.2.10",
      "port": 22,
      "user": "ubuntu",
      "vars": {
        "onigirazu_host": "10.0.2.10",
        "instance_type": "r5.large",
        "cloud_provider": "aws",
        "region": "us-east-1",
        "db_engine": "postgresql"
      }
    }
  ],
  "groups": {
    "aws_instances": {
      "name": "aws_instances",
      "hosts": {
        "dynamic-web1": {},
        "dynamic-web2": {},
        "dynamic-db1": {}
      },
      "vars": {
        "cloud_provider": "aws",
        "managed_by": "terraform"
      },
      "children": []
    },
    "web_tier": {
      "name": "web_tier",
      "hosts": {
        "dynamic-web1": {},
        "dynamic-web2": {}
      },
      "vars": {
        "tier": "web",
        "auto_scaling": true
      },
      "children": []
    },
    "db_tier": {
      "name": "db_tier",
      "hosts": {
        "dynamic-db1": {}
      },
      "vars": {
        "tier": "database",
        "backup_enabled": true
      },
      "children": []
    }
  }
}
EOF
elif [ "$1" = "--host" ]; then
    echo '{"_meta": {"hostvars": {}}}'
else
    echo "Usage: $0 --list|--host <hostname>"
    exit 1
fi
