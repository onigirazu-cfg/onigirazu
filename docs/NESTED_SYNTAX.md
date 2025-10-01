# Nested Module Syntax in Onigirazu

Onigirazu поддерживает два различных синтаксиса для определения модулей в задачах:

## 1. Flat Syntax (текущий стандарт)

```yaml
- name: "Install package"
  module: "package"
  name: "tree"
  state: "present"
```

## 2. Nested Syntax (новый рекомендуемый)

```yaml
- name: "Install package"
  module:
    type: "package"
    name: "tree"
    state: "present"
```

## Преимущества вложенного синтаксиса

### Логическая группировка

Все параметры модуля логически сгруппированы под блоком `module:`, что делает структуру более понятной.

### Четкое разделение

Параметры модуля четко отделены от параметров задачи (таких как `name`, `ignore_errors`, `when` и т.д.).

### Лучшая читаемость

```yaml
# Вложенный синтаксис - четко видно, что относится к модулю
- name: "Complex task with conditions"
  module:
    type: "command"
    command: "systemctl restart nginx"
    creates: "/var/run/nginx.pid"
  when: "nginx_restart_needed"
  ignore_errors: true
  notify: "check nginx status"

# vs плоский синтаксис - сложнее различить параметры
- name: "Complex task with conditions"
  module: "command"
  command: "systemctl restart nginx"
  creates: "/var/run/nginx.pid"
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
    command: "uname -a"
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

## Обратная совместимость

Оба синтаксиса полностью поддерживаются и могут использоваться в одном playbook:

```yaml
plays:
  - name: "Mixed syntax example"
    hosts: servers
    tasks:
      # Flat syntax
      - name: "Task 1"
        module: "command"
        command: "echo 'flat'"

      # Nested syntax (рекомендуемый)
      - name: "Task 2"
        module:
          type: "command"
          command: "echo 'nested'"
```

## Установка вложенного синтаксиса как правила

### Через конфигурационный файл

Создайте файл `onigirazu.yml` в корне проекта:

```yaml
# Предпочтительный синтаксис (рекомендация)
preferred_module_syntax: "nested"
enforce_module_syntax: false

# Строгий режим (только вложенный синтаксис)
preferred_module_syntax: "nested"
enforce_module_syntax: true
```

### Через переменные окружения

```bash
# Установить вложенный синтаксис как предпочтительный
export ONIGIRAZU_PREFERRED_MODULE_SYNTAX="nested"

# Принудительно требовать только вложенный синтаксис
export ONIGIRAZU_ENFORCE_MODULE_SYNTAX="true"
```

### Примеры конфигураций

**Мягкий режим** (рекомендация, но разрешены оба синтаксиса):

```yaml
preferred_module_syntax: "nested"
enforce_module_syntax: false
```

**Строгий режим** (только вложенный синтаксис):

```yaml
preferred_module_syntax: "nested"
enforce_module_syntax: true
```

## Рекомендации

1. **Для новых проектов** - используйте строгий режим с вложенным синтаксисом
2. **Для существующих проектов** - начните с мягкого режима и постепенно мигрируйте
3. **Для команд** - установите единый стандарт через конфигурацию
4. **Для CI/CD** - используйте строгий режим для обеспечения консистентности

## Техническая реализация

Поддержка вложенного синтаксиса реализована в методе `UnmarshalYAML` структуры `Task` в файле `pkg/types/types.go`. Парсер автоматически определяет тип синтаксиса и корректно извлекает параметры модуля.
