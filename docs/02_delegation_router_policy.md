# Delegation Router Policy (L7 AI Proxy)

Роутер действует как API Gateway, маршрутизирующий задачи на основе быстрого статического анализа и вычисленного State Vector ($\vec{S}_k$).

## Decision Tree

Делегирование происходит в Go без участия LLM для снижения latency до $T_{route} < 50ms$ и минимизации рисков компрометации (LLM prompt injection для угона роутинга).

### Phase 1: Security & Math Gate (Deterministic)
```go
if regex.Match(`[\$\\]\w+|proof|verify|sym|integral`, prompt) || db.HasCryptoKeywords(prompt) {
    // Requires exactness. No hallucinations allowed.
    return DelegateTo(MathVerifierAgent) // Sandbox: Python AST + SymPy execution
}
```

### Phase 2: Complexity Heuristic Score ($H_{score}$)
$H_{score}$ вычисляется как взвешенная сумма:
- $H_1$: Длина символов (Length / 1000)
- $H_2$: Плотность кода (Символы `{`, `}`, `def`, `func` / Total Length)
- $H_3$: Глубина архитектурных терминов (refactor, microservices, consensus)

```go
if H_score >= Threshold.HeavyCompute {
    return DelegateTo(Claude_4_6_Opus) // Long reasoning, complex code rewriting
} else {
    return DelegateTo(Gemini_3_1_Pro) // Tool usage, chat, immediate exec
}
```

## Anti-Vibecode Rules for Implementation
1. Роутер не должен быть LLM-агентом. Только статика и алгоритмы. 
2. Никаких stateful connections в роутере. Весь state в PostgreSQL.
3. Обработка таймаутов $T > 10s$ с fallback на Gemini Pro.
