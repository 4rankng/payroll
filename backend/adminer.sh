#!/bin/bash

# Adminer Access Control Script
# Usage: ./adminer.sh --on|--off|--status

SERVER="pay.1stop.app"
ADMINER_PORT="8081"

show_usage() {
    echo "Usage: $0 [--on|--off|--status]"
    echo "  --on      Expose Adminer to internet (port $ADMINER_PORT)"
    echo "  --off     Hide Adminer from internet"
    echo "  --status  Show current Adminer status"
    exit 1
}

check_connection() {
    echo "Checking SSH connection to $SERVER..."
    if ! ssh -o ConnectTimeout=5 root@$SERVER "echo 'Connected'" > /dev/null 2>&1; then
        echo "❌ Error: Cannot connect to $SERVER"
        echo "Please ensure SSH access is working"
        exit 1
    fi
}

enable_adminer() {
    echo "🔓 Enabling Adminer access..."

    # Allow port in firewall
    ssh root@$SERVER "ufw allow $ADMINER_PORT comment 'Adminer Database Admin'" 2>/dev/null

    # Check if already running with port exposed
    CURRENT_PORTS=$(ssh root@$SERVER "docker port payroll-adminer 2>/dev/null" | grep "$ADMINER_PORT")

    if [[ -n "$CURRENT_PORTS" ]]; then
        echo "✅ Adminer is already accessible at http://$SERVER:$ADMINER_PORT"
        return
    fi

    echo "Restarting Adminer with public port exposure..."
    ssh root@$SERVER "
        docker stop payroll-adminer > /dev/null 2>&1
        docker rm payroll-adminer > /dev/null 2>&1
        docker run -d \
            --name payroll-adminer \
            --restart unless-stopped \
            -p $ADMINER_PORT:8080 \
            --network root_default \
            -e ADMINER_DEFAULT_SERVER=payroll-mysql \
            adminer:latest
    "

    sleep 3

    # Verify it's working
    sleep 3
    if ssh root@$SERVER "curl -s --max-time 5 http://localhost:$ADMINER_PORT" > /dev/null; then
        echo "✅ Adminer is now accessible at: http://$SERVER:$ADMINER_PORT"
        echo "📋 Default login:"
        echo "   Server: payroll-mysql"
        echo "   Username: payroll_user"
        echo "   Password: payroll_pass"
        echo "   Database: payroll_db"
    else
        echo "❌ Error: Adminer failed to start properly"
        exit 1
    fi
}

disable_adminer() {
    echo "🔒 Disabling Adminer access..."

    # Block port in firewall
    ssh root@$SERVER "ufw delete allow $ADMINER_PORT > /dev/null 2>&1 || true"
    ssh root@$SERVER "ufw deny $ADMINER_PORT comment 'Block Adminer external access'" 2>/dev/null

    echo "Restarting Adminer without public port exposure..."
    ssh root@$SERVER "
        docker stop payroll-adminer > /dev/null 2>&1
        docker rm payroll-adminer > /dev/null 2>&1
        docker run -d \
            --name payroll-adminer \
            --restart unless-stopped \
            --network root_default \
            -e ADMINER_DEFAULT_SERVER=payroll-mysql \
            adminer:latest
    "

    sleep 2

    # Verify it's blocked
    sleep 2
    if ! ssh root@$SERVER "curl -s --max-time 3 http://localhost:$ADMINER_PORT" > /dev/null 2>&1; then
        echo "✅ Adminer is now hidden from internet access"
        echo "🔒 Access blocked on port $ADMINER_PORT"
    else
        echo "⚠️  Warning: Adminer might still be accessible"
    fi
}

show_status() {
    echo "📊 Current Adminer Status:"

    # Check firewall status
    FIREWALL_STATUS=$(ssh root@$SERVER "ufw status | grep $ADMINER_PORT" 2>/dev/null)
    if echo "$FIREWALL_STATUS" | grep -q "ALLOW"; then
        echo "🔓 Firewall: ALLOWED"
    elif echo "$FIREWALL_STATUS" | grep -q "DENY"; then
        echo "🔒 Firewall: BLOCKED"
    else
        echo "❓ Firewall: NOT CONFIGURED"
    fi

    # Check container status
    CONTAINER_STATUS=$(ssh root@$SERVER "docker ps --filter name=payroll-adminer --format '{{.Status}}'" 2>/dev/null)
    if [[ -n "$CONTAINER_STATUS" ]]; then
        echo "🐳 Container: Running ($CONTAINER_STATUS)"

        # Check port exposure
        PORT_STATUS=$(ssh root@$SERVER "docker port payroll-adminer" 2>/dev/null)
        if echo "$PORT_STATUS" | grep -q "$ADMINER_PORT"; then
            echo "🌐 Port: EXPOSED ($(echo "$PORT_STATUS" | grep "$ADMINER_PORT"))"
        else
            echo "🔒 Port: INTERNAL ONLY"
        fi
    else
        echo "🐳 Container: NOT RUNNING"
    fi

    # Test external access
    echo -n "🌍 External Access: "
    if ssh root@$SERVER "curl -s --max-time 3 http://localhost:$ADMINER_PORT" > /dev/null 2>&1; then
        echo "ACCESSIBLE at http://$SERVER:$ADMINER_PORT"
    else
        echo "BLOCKED"
    fi
}

# Main script logic
case "$1" in
    --on)
        check_connection
        enable_adminer
        ;;
    --off)
        check_connection
        disable_adminer
        ;;
    --status)
        check_connection
        show_status
        ;;
    *)
        show_usage
        ;;
esac