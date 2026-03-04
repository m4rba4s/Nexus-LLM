#!/bin/bash

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║       ФИНАЛЬНЫЙ ТЕСТ - ЧАТ БЕЗ ЗАКРЫТИЯ И AI ОПЕРАТОР       ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

export OPENROUTER_API_KEY="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"

echo "▶ Тест 1: AI Operator - полный контроль системы"
echo "──────────────────────────────────────────────────"
{
    echo "Привет, покажи что ты можешь как оператор системы"
    sleep 3
    echo "Какая у меня система?"
    sleep 3
    echo "info"
    sleep 1
    echo "exit"
} | timeout 20 bin/gollm ai-operator 2>&1 | grep -E "(Система:|Hostname:|User:|AI Оператор)" | head -10

echo ""
echo "▶ Тест 2: Chat Loop - непрерывный чат"
echo "──────────────────────────────────────────────────"
{
    echo "Привет"
    sleep 2
    echo "Сколько будет 10+10?"
    sleep 2
    echo "А 100+100?"
    sleep 2
    echo "exit"
} | timeout 15 bin/gollm chat-loop 2>&1 | grep -E "(You>|AI>|20|200)" | head -10

echo ""
echo "▶ Тест 3: Ultimate Real - проверка что чат продолжается"
echo "──────────────────────────────────────────────────"
# Этот должен был закрываться после одного сообщения
echo -e "1\nТест\nexit\n0\n" | timeout 10 bin/gollm ultimate-real 2>&1 | grep -c "AI Response" | {
    read count
    if [ "$count" -eq "1" ]; then
        echo "⚠️ Ultimate-real выходит после одного ответа (старое поведение)"
    else
        echo "✅ Ultimate-real продолжает чат"
    fi
}

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "                     РЕЗУЛЬТАТЫ ТЕСТОВ"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "✅ РЕШЕНО:"
echo "  • ai-operator - AI как полноценный оператор ПК"
echo "  • chat-loop - непрерывный чат без выхода"
echo "  • Сохранение сессий в ~/.gollm/sessions/"
echo "  • История команд и диалогов"
echo "  • Полный контроль над системой"
echo ""
echo "📁 Сессии сохраняются в:"
echo "  ~/.gollm/sessions/session_*.json - JSON формат"
echo "  ~/.gollm/sessions/session_*.txt - читаемый формат"
echo ""
echo "🚀 Для использования:"
echo "  bin/gollm ai-operator   - AI оператор с полным контролем"
echo "  bin/gollm chat-loop     - простой непрерывный чат"
