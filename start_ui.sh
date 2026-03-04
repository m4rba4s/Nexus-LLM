#!/bin/bash

# GOLLM Advanced UI Launcher
# Запускает супер крутой интерфейс для работы с LLM

clear

# Цвета
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${CYAN}"
cat << "EOF"
╔══════════════════════════════════════════════════════════════════════╗
║   ██████╗  ██████╗ ██╗     ██╗     ███╗   ███╗                     ║
║  ██╔════╝ ██╔═══██╗██║     ██║     ████╗ ████║                     ║
║  ██║  ███╗██║   ██║██║     ██║     ██╔████╔██║                     ║
║  ██║   ██║██║   ██║██║     ██║     ██║╚██╔╝██║                     ║
║  ╚██████╔╝╚██████╔╝███████╗███████╗██║ ╚═╝ ██║                     ║
║   ╚═════╝  ╚═════╝ ╚══════╝╚══════╝╚═╝     ╚═╝ PIZDA VSEM KRYSAM    "|                                                                    ║
║          🚀 ADVANCED AI TERMINAL INTERFACE 🤖                       ║
╚══════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${GREEN}Добро пожаловать в GOLLM Advanced Interface!${NC}"
echo -e "${PURPLE}Самый крутой терминальный интерфейс для работы с AI${NC}\n"

echo -e "${YELLOW}Выберите режим запуска:${NC}\n"
echo -e "  ${CYAN}1)${NC} 🎨 Neon Theme (по умолчанию)"
echo -e "  ${CYAN}2)${NC} 🌌 Matrix Theme"
echo -e "  ${CYAN}3)${NC} 🌙 Dark Theme"
echo -e "  ${CYAN}4)${NC} ☀️  Light Theme"
echo -e "  ${CYAN}5)${NC} ⚡ С автовыполнением команд"
echo -e "  ${CYAN}6)${NC} 🔧 С MCP сервером"
echo -e "  ${CYAN}7)${NC} 🚀 Полная конфигурация (MCP + Auto)"
echo -e "  ${CYAN}8)${NC} 📖 Показать справку"
echo -e "  ${CYAN}9)${NC} 🔙 Выход\n"

read -p "Ваш выбор (1-9): " choice

case $choice in
    1)
        echo -e "\n${GREEN}Запуск с темой Neon...${NC}"
        ./bin/gollm advanced --theme neon
        ;;
    2)
        echo -e "\n${GREEN}Запуск с темой Matrix...${NC}"
        ./bin/gollm advanced --theme matrix
        ;;
    3)
        echo -e "\n${GREEN}Запуск с темой Dark...${NC}"
        ./bin/gollm advanced --theme dark
        ;;
    4)
        echo -e "\n${GREEN}Запуск с темой Light...${NC}"
        ./bin/gollm advanced --theme light
        ;;
    5)
        echo -e "\n${GREEN}Запуск с автовыполнением команд...${NC}"
        echo -e "${YELLOW}⚠️  Будьте осторожны! Команды будут выполняться автоматически${NC}"
        ./bin/gollm advanced --auto-execute
        ;;
    6)
        echo -e "\n${GREEN}Запуск с MCP сервером...${NC}"
        read -p "Введите порт для MCP (по умолчанию 8080): " port
        port=${port:-8080}
        ./bin/gollm advanced --mcp-port $port
        ;;
    7)
        echo -e "\n${GREEN}Запуск с полной конфигурацией...${NC}"
        echo -e "${YELLOW}MCP сервер + Автовыполнение команд${NC}"
        ./bin/gollm advanced --mcp-port 8080 --auto-execute --theme neon
        ;;
    8)
        echo -e "\n${CYAN}═══════════════════════════════════════════${NC}"
        echo -e "${GREEN}СПРАВКА ПО ИСПОЛЬЗОВАНИЮ${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════${NC}\n"
        
        echo -e "${YELLOW}Горячие клавиши:${NC}"
        echo -e "  Ctrl+M - Открыть главное меню"
        echo -e "  Ctrl+E - Вкл/выкл автовыполнение"
        echo -e "  Ctrl+C - Выход"
        echo -e "  ↑/↓    - Навигация"
        echo -e "  Enter  - Отправить/выбрать"
        
        echo -e "\n${YELLOW}Команды в чате:${NC}"
        echo -e "  /help     - Показать справку"
        echo -e "  /clear    - Очистить историю"
        echo -e "  /run <cmd> - Выполнить команду"
        echo -e "  /mcp      - Показать MCP инструменты"
        echo -e "  /theme    - Сменить тему"
        
        echo -e "\n${YELLOW}MCP инструменты:${NC}"
        echo -e "  📁 Работа с файлами"
        echo -e "  🌐 HTTP запросы"
        echo -e "  💾 База данных"
        echo -e "  🔍 Поиск и анализ кода"
        echo -e "  📊 Обработка данных"
        
        echo -e "\n${PURPLE}Нажмите Enter для возврата в меню...${NC}"
        read
        exec $0
        ;;
    9)
        echo -e "\n${GREEN}До свидания! 👋${NC}"
        exit 0
        ;;
    *)
        echo -e "\n${YELLOW}Запуск с настройками по умолчанию...${NC}"
        ./bin/gollm advanced
        ;;
esac
