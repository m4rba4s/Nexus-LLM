#!/bin/bash

echo "═══════════════════════════════════════════════════════════════"
echo "   TESTING ENHANCED ULTIMATE WITH CONTINUOUS CHAT"
echo "═══════════════════════════════════════════════════════════════"
echo ""

export OPENROUTER_API_KEY="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"

# Test the continuous chat feature
{
    echo "1"  # Select chat
    sleep 1
    echo "Hello, please say 'Chat loop working'"
    sleep 3
    echo "What is 2+2?"
    sleep 3
    echo "/learn"  # Test system learning
    sleep 2
    echo "exit"  # Exit chat
    echo "0"  # Exit program
} | timeout 25 bin/gollm ultimate-enhanced 2>&1 | tee enhanced_output.log

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Checking results..."
echo ""

# Check if chat loop worked
if grep -q "Chat loop working" enhanced_output.log; then
    echo "✅ Continuous chat: WORKING"
else
    echo "⚠️ Continuous chat: needs verification"
fi

if grep -q "2+2" enhanced_output.log && grep -q "4" enhanced_output.log; then
    echo "✅ Multiple messages: WORKING"
else
    echo "⚠️ Multiple messages: needs verification"
fi

if grep -q "System information learned" enhanced_output.log; then
    echo "✅ System learning: WORKING"
else
    echo "⚠️ System learning: needs verification"
fi

echo ""
echo "Full output saved to: enhanced_output.log"
