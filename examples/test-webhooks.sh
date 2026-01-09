#!/bin/bash

# Test script for k8flex webhook integrations
# Tests all supported alerting systems with example payloads

set -e

K8FLEX_URL="${K8FLEX_URL:-http://localhost:8080/webhook}"
EXAMPLES_DIR="$(dirname "$0")/webhooks"

echo "Testing k8flex webhook integrations..."
echo "URL: $K8FLEX_URL"
echo "Examples: $EXAMPLES_DIR"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_webhook() {
    local name=$1
    local file=$2
    
    echo -n "Testing $name... "
    
    if [ ! -f "$file" ]; then
        echo -e "${RED}FAILED${NC} - File not found: $file"
        return 1
    fi
    
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d @"$file" \
        "$K8FLEX_URL" 2>&1)
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}OK${NC}"
        echo "  Response: $body"
        return 0
    else
        echo -e "${RED}FAILED${NC} (HTTP $http_code)"
        echo "  Response: $body"
        return 1
    fi
}

echo "================================"
echo "Testing Alerting System Webhooks"
echo "================================"
echo ""

# Test each integration
test_webhook "Alertmanager" "$EXAMPLES_DIR/alertmanager.json" || true
test_webhook "PagerDuty" "$EXAMPLES_DIR/pagerduty.json" || true
test_webhook "Grafana" "$EXAMPLES_DIR/grafana.json" || true
test_webhook "Datadog" "$EXAMPLES_DIR/datadog.json" || true
test_webhook "Opsgenie" "$EXAMPLES_DIR/opsgenie.json" || true
test_webhook "VictorOps" "$EXAMPLES_DIR/victorops.json" || true
test_webhook "New Relic" "$EXAMPLES_DIR/newrelic.json" || true

echo ""
echo "================================"
echo "Test Summary"
echo "================================"
echo ""
echo "Check k8flex logs for processing details:"
echo "  kubectl logs -n k8flex deployment/k8flex-agent --tail=50"
echo ""
echo "Or follow logs in real-time:"
echo "  kubectl logs -n k8flex deployment/k8flex-agent -f"
