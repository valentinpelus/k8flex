#!/bin/bash
# Database Setup Script for k8flex Knowledge Base
# This script initializes a PostgreSQL database with pgvector extension

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-k8flex}"
DB_USER="${DB_USER:-k8flex}"
DB_PASSWORD="${DB_PASSWORD:-changeme}"

echo "🚀 Setting up k8flex Knowledge Base"
echo "=================================="
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo "User: $DB_USER"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "❌ Error: psql is not installed"
    echo "Please install PostgreSQL client:"
    echo "  macOS: brew install postgresql"
    echo "  Ubuntu: apt-get install postgresql-client"
    exit 1
fi

# Test database connection
echo "🔍 Testing database connection..."
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c '\q' 2>/dev/null; then
    echo "❌ Error: Cannot connect to PostgreSQL"
    echo "Please check your connection settings:"
    echo "  DB_HOST=$DB_HOST"
    echo "  DB_PORT=$DB_PORT"
    echo "  DB_USER=$DB_USER"
    exit 1
fi
echo "✅ Connection successful"

# Check if database exists, create if not
echo ""
echo "🔍 Checking database..."
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo "✅ Database '$DB_NAME' already exists"
else
    echo "📦 Creating database '$DB_NAME'..."
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;"
    echo "✅ Database created"
fi

# Run migrations
echo ""
echo "📋 Running migrations..."
MIGRATION_FILE="$(dirname "$0")/../deployments/migrations/001_init_knowledge_base.sql"

if [ ! -f "$MIGRATION_FILE" ]; then
    echo "❌ Error: Migration file not found: $MIGRATION_FILE"
    exit 1
fi

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$MIGRATION_FILE"

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Knowledge base initialized successfully!"
    echo ""
    echo "📊 Database Statistics:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT * FROM alert_cases_stats;"
    
    echo ""
    echo "🔗 Connection String:"
    echo "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=prefer"
    echo ""
    echo "📝 Next Steps:"
    echo "1. Update your Helm values.yaml with the database URL"
    echo "2. Configure the embedding provider (OpenAI or Gemini)"
    echo "3. Deploy k8flex with knowledgeBase.enabled: true"
else
    echo "❌ Migration failed"
    exit 1
fi
