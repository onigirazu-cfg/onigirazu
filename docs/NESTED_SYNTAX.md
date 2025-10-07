# Module Syntax in Onigirazu

Onigirazu поддерживает два стиля синтаксиса для определения модулей в задачах:

## 1. Expanded Syntax (рекомендуемый для сложных задач)

```yaml
- name: "Install package"
  module:
    type: "package"
    name: "tree"
    state: "present"
```

## 2. Inline Syntax (удобный для простых задач)

```yaml
- name: "Install package"
  module: { type: "package", name: "tree", state: "present" }
```

## Преимущества expanded синтаксиса

### Логическая группировка

Все параметры модуля логически сгруппированы под блоком `module:`, что делает структуру более понятной.

### Четкое разделение

Параметры модуля четко отделены от параметров задачи (таких как `name`, `ignore_errors`, `when` и т.д.).

### Лучшая читаемость

```yaml
# Expanded синтаксис - четко видно, что относится к модулю
- name: "Complex task with conditions"
  module:
    type: "command"
    cmd: "systemctl restart nginx"
    creates: "/var/run/nginx.pid"
  when: "nginx_restart_needed"
  ignore_errors: true
  notify: "check nginx status"

# vs Inline синтаксис - компактнее, но менее читаемый для сложных задач
- name: "Complex task with conditions"
  module: { type: "command", cmd: "systemctl restart nginx", creates: "/var/run/nginx.pid" }
  when: "nginx_restart_needed"
  ignore_errors: true
  notify: "check nginx status"
```

## Примеры использования

### Command модуль

```yaml
- name: "Check system info"
  module:
    type: "command"
    cmd: "uname -a"
```

### Package модуль

```yaml
- name: "Install tree package"
  module:
    type: "package"
    name: "tree"
    state: "present"
```

### File модуль

```yaml
- name: "Create config file"
  module:
    type: "file"
    path: "/etc/myapp/config.yml"
    content: "{{ config_template }}"
    mode: "0644"
    owner: "root"
    group: "root"
```

### Service модуль

```yaml
- name: "Start nginx service"
  module:
    type: "service"
    name: "nginx"
    state: "started"
    enabled: true
```

## Гибкость использования

Оба стиля синтаксиса могут использоваться в одном playbook:

```yaml
plays:
  - name: "Mixed syntax example"
    hosts: servers
    tasks:
      # Inline syntax - для простых задач
      - name: "Task 1"
        module: { type: "command", cmd: "echo 'inline'" }

      # Expanded syntax - для сложных задач (рекомендуемый)
      - name: "Task 2"
        module:
          type: "command"
          cmd: "echo 'expanded'"
```

## Рекомендации по выбору стиля

### Используйте Expanded Syntax когда

1. **Задача имеет много параметров** (более 3-4)
2. **Параметры содержат сложные значения** (многострочные строки, списки, объекты)
3. **Нужна максимальная читаемость** (для документации, обучения)
4. **Задача критична** и требует тщательного review

### Используйте Inline Syntax когда

1. **Задача простая** (1-3 параметра)
2. **Все значения короткие** (простые строки, числа, булевы значения)
3. **Нужна компактность** (много похожих простых задач)
4. **Задача очевидна** и не требует детального изучения

## Примеры выбора стиля

### ✅ Хороший выбор Inline

```yaml
- name: "Install git"
  module: { type: "package", name: "git", state: "present" }

- name: "Check uptime"
  module: { type: "command", cmd: "uptime" }
```

### ✅ Хороший выбор Expanded

```yaml
- name: "Deploy application configuration"
  module:
    type: "copy"
    dest: "/etc/app/config.yml"
    content: |
      server:
        host: 0.0.0.0
        port: 8080
      database:
        host: db.example.com
        port: 5432
    mode: "0644"
    owner: "app"
    group: "app"
    backup: true
```

## Техническая реализация

Оба стиля синтаксиса поддерживаются парсером YAML в методе `UnmarshalYAML` структуры `Task` в файле `pkg/types/types.go`. Парсер автоматически определяет стиль синтаксиса и корректно извлекает параметры модуля.
