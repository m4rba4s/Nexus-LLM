# State Machine Kinematics (Emotion Engine)

Симуляция "эмоций" без промптов требует детерминированной математической модели. Промпты уязвимы к injection и галлюцинациям. Мы переводим состояния в векторы и матрицы, вычисляемые Go-ядром.

## State Vector Definition

Вектор состояния системы на шаге $k$:
$$ \vec{S}_k = \begin{bmatrix} P \\ C \\ F \end{bmatrix}_k $$
Где:
*   $P \in [0, 1]$: **Paranoia** (Паранойя)
*   $C \in [0, 1]$: **Curiosity** (Любопытство)
*   $F \in [0, 1]$: **Frustration** (Фрустрация)

## State Transition Equation (LTI System Approximation)

Обновление вектора при каждом входящем событии:
$$ \vec{S}_k = Max(0, Min(1, A \cdot \vec{S}_{k-1} + B \cdot \vec{U}_k)) $$

### Matrix A (Decay / Forgetting Curve)
Матрица затухания эмоций во времени, если нет подкрепляющих триггеров:
$$ A = \begin{bmatrix} e^{-\lambda_p \Delta t} & 0 & 0 \\ 0 & e^{-\lambda_c \Delta t} & 0 \\ 0 & 0 & e^{-\lambda_f \Delta t} \end{bmatrix} $$

### Matrix B (Sensitivity Weights)
Матрица чувствительности системы к конкретным триггерам:
$$ B = \begin{bmatrix} W_{p\_err} & W_{p\_sec} & 0 \\ 0 & W_{c\_new} & W_{c\_complex} \\ W_{f\_rep} & 0 & W_{f\_timeout} \end{bmatrix} $$

### Input Vector ($\vec{U}_k$)
Вектор триггеров, собранных на этапе пре-процессинга конкретного запроса:
*   $u_1$: Количество синтаксических ошибок (или retry).
*   $u_2$: Наличие security-слов (exploit, payload) - $BOOL \to [0, 1]$.
*   $u_3$: Новизна библиотек (поиск по локальному кэшу AST).
*   $u_4$: Репетативность пользователя (cosine similarity последних 5 запросов).
*   $u_5$: Ошибки API/Таймауты моделей.

## Routing Impact (Policy)

Значение $\vec{S}_k$ напрямую влияет на пайплайн до того, как запрос уйдёт в LLM:
1.  **If $P \ge 0.75$**: Включается `StrictASTAnalysis = true`, запрос проходит через фаервол регулярных выражений, контекст ограничивается.
2.  **If $F \ge 0.60$**: Системный промпт инжектируется флагом *Mode=CynicalDirect*. Длина ответа $\lim L_{max} = 150$ токенов.
3.  **If $C \ge 0.80$**: Повышается `temperature` на +0.2, включается RAG для поиска нестандартных паттернов по документации.
