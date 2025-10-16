# FAQ: insecure_ignore_host_key

## Frequently Asked Questions

### Q: Як можна передати `insecure_ignore_host_key`?

**A:** Параметр `insecure_ignore_host_key` передається через **inventory файл**, а не через playbook або аргументи модуля.

**✅ Правильно:**

```yaml
# inventory.yml
hosts:
  myhost:
    address: 192.168.1.10
    insecure_ignore_host_key: true  # ← Тут!
```

**❌ Неправильно:**

```yaml
# playbook.yml
tasks:
  - name: Run command
    shell:
      command: hostname
      insecure_ignore_host_key: true  # ← Це не працює!
```

---

### Q: Чи можна використовувати `insecure_ignore_host_key` у текстовому інвенторі?

**A:** **НІ**, текстовий формат (`.txt`, `.ini`) **не підтримує** цей параметр.

**❌ Не працює:**

```text
# inventory.txt
192.168.1.10
192.168.1.11:2222
```

**✅ Рішення:** Використовуйте YAML, TOML або JSON:

```yaml
# inventory.yml
hosts:
  server-01:
    address: 192.168.1.10
    insecure_ignore_host_key: true
```

Детальніше: [Text Format Limitations](./inventory_text_format_limitations.md)

---

### Q: Які формати інвентору підтримують `insecure_ignore_host_key`?

**A:** Тільки структуровані формати:

| Формат | Підтримка | Рекомендація |
|--------|-----------|--------------|
| YAML (`.yml`, `.yaml`) | ✅ | **Рекомендовано** |
| TOML (`.toml`) | ✅ | Добре |
| JSON (`.json`) | ✅ | Добре |
| Text (`.txt`, `.ini`) | ❌ | Не підтримується |

---

### Q: Як встановити `insecure_ignore_host_key` для всіх хостів?

**A:** Використовуйте групу `all`:

```yaml
groups:
  all:
    vars:
      insecure_ignore_host_key: true  # Застосується до ВСІХ хостів

hosts:
  server-01:
    address: 192.168.1.10
  server-02:
    address: 192.168.1.11
  # Обидва хости використовують insecure_ignore_host_key: true
```

---

### Q: Як встановити для окремого хоста?

**A:** Додайте параметр безпосередньо в конфігурацію хоста:

```yaml
hosts:
  dev-server:
    address: 192.168.1.10
    insecure_ignore_host_key: true  # Тільки цей хост

  prod-server:
    address: 10.0.1.100
    # insecure_ignore_host_key не встановлено = безпечний режим (за замовчуванням)
```

---

### Q: Як встановити для групи хостів?

**A:** Використовуйте змінні групи:

```yaml
groups:
  dev-servers:
    hosts:
      - dev-01
      - dev-02
    vars:
      insecure_ignore_host_key: true  # Всі хости в цій групі

  prod-servers:
    hosts:
      - prod-01
      - prod-02
    # insecure_ignore_host_key не встановлено = безпечний режим

hosts:
  dev-01:
    address: 192.168.1.10
  dev-02:
    address: 192.168.1.11
  prod-01:
    address: 10.0.1.100
  prod-02:
    address: 10.0.1.101
```

---

### Q: Який пріоритет налаштувань?

**A:** Пріоритет (від вищого до нижчого):

1. **Host-level** (налаштування хоста)
2. **Group-level** (налаштування групи)
3. **All-level** (налаштування групи `all`)

**Приклад:**

```yaml
hosts:
  server:
    address: 192.168.1.10
    insecure_ignore_host_key: false  # ← Це виграє! (найвищий пріоритет)

groups:
  mygroup:
    hosts:
      - server
    vars:
      insecure_ignore_host_key: true  # ← Ігнорується

  all:
    vars:
      insecure_ignore_host_key: true  # ← Ігнорується
```

Результат: `server` використовує **безпечний режим** (host-level має пріоритет).

---

### Q: Чи потрібно щось змінювати в коді модуля?

**A:** **НІ!** Якщо ви використовуєте `BaseExecutorModule`, параметр автоматично застосовується:

```go
type MyModule struct {
    *modules.BaseExecutorModule
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // host.InsecureIgnoreHostKey вже встановлено з inventory

    // Всі ці методи автоматично використовують host.InsecureIgnoreHostKey:
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })

    // Нічого додаткового робити не потрібно!
    return result, nil
}
```

---

### Q: Чи безпечно використовувати `insecure_ignore_host_key: true`?

**A:** **НІ!** Це небезпечно і робить вас вразливими до MITM атак.

**✅ Використовуйте ТІЛЬКИ для:**

- Локальної розробки (Vagrant, Docker)
- Тестових середовищ
- CI/CD з динамічними хостами
- Ефемерних контейнерів

**❌ НІКОЛИ не використовуйте для:**

- Продакшн серверів
- Staging середовищ
- Серверів з реальними даними
- Публічної інфраструктури

---

### Q: Як виправити "Host key verification failed"?

**A:** Є три способи:

**1. Для розробки (швидко, але небезпечно):**

```yaml
# inventory.yml
hosts:
  myhost:
    insecure_ignore_host_key: true
```

**2. Для продакшн (правильно):**

```bash
# Додати ключ хоста до known_hosts
ssh-keyscan -H hostname >> ~/.ssh/known_hosts
```

**3. Якщо ключ змінився:**

```bash
# Видалити старий ключ
ssh-keygen -R hostname
# Додати новий ключ
ssh-keyscan -H hostname >> ~/.ssh/known_hosts
```

---

### Q: Як мігрувати з текстового формату на YAML?

**A:** Дивіться детальний гайд: [Text Format Limitations](./inventory_text_format_limitations.md)

**Швидка міграція:**

**До (inventory.txt):**

```text
192.168.1.10
192.168.1.11:2222
deploy@192.168.1.20
```

**Після (inventory.yml):**

```yaml
groups:
  all:
    vars:
      insecure_ignore_host_key: true  # Застосувати до всіх

hosts:
  host-1:
    address: 192.168.1.10
    user: root
    port: 22

  host-2:
    address: 192.168.1.11
    user: root
    port: 2222

  host-3:
    address: 192.168.1.20
    user: deploy
    port: 22
```

Простий приклад: [inventory_simple_with_insecure.yml](./inventory_simple_with_insecure.yml)

---

### Q: Як перевірити, чи працює налаштування?

**A:** Увімкніть debug логування:

```bash
onigirazu-cli run -i inventory.yml playbook.yml --log-level debug
```

Шукайте в логах:

```
DEBUG: Host 'myhost' InsecureIgnoreHostKey: true
DEBUG: Skipping host key verification for host 'myhost'
```

---

### Q: Чи можна використовувати різні налаштування для різних середовищ?

**A:** Так! Використовуйте окремі inventory файли:

```bash
inventory/
  ├── dev.yml          # insecure_ignore_host_key: true
  ├── staging.yml      # insecure_ignore_host_key: false
  └── production.yml   # insecure_ignore_host_key: false (або не встановлено)
```

**Використання:**

```bash
# Розробка
onigirazu-cli run -i inventory/dev.yml playbook.yml

# Staging
onigirazu-cli run -i inventory/staging.yml playbook.yml

# Продакшн
onigirazu-cli run -i inventory/production.yml playbook.yml
```

---

### Q: Що таке значення за замовчуванням?

**A:** За замовчуванням `insecure_ignore_host_key: false` (безпечний режим).

Це означає, що SSH перевіряє ключі хостів через `~/.ssh/known_hosts`.

---

### Q: Де знайти більше інформації?

**A:** Дивіться документацію:

- **Повний гайд:** [README_insecure_ignore_host_key.md](./README_insecure_ignore_host_key.md)
- **Швидкий довідник:** [QUICK_REFERENCE_insecure_ignore_host_key.md](./QUICK_REFERENCE_insecure_ignore_host_key.md)
- **Візуальна діаграма:** [FLOW_insecure_ignore_host_key.md](./FLOW_insecure_ignore_host_key.md)
- **Обмеження текстового формату:** [inventory_text_format_limitations.md](./inventory_text_format_limitations.md)
- **Приклад inventory:** [inventory_with_insecure_host_key.yml](./inventory_with_insecure_host_key.yml)
- **Простий приклад:** [inventory_simple_with_insecure.yml](./inventory_simple_with_insecure.yml)
- **Приклад playbook:** [playbook_with_insecure_hosts.yml](./playbook_with_insecure_hosts.yml)
- **Приклад модуля:** [example_module_with_base_executor.go](./example_module_with_base_executor.go)

---

## Швидкий старт

**1. Створіть YAML inventory:**

```yaml
# inventory.yml
groups:
  all:
    vars:
      insecure_ignore_host_key: true  # ⚠️ Тільки для dev/test!

hosts:
  myhost:
    address: 192.168.1.10
```

**2. Використовуйте його:**

```bash
onigirazu-cli run -i inventory.yml playbook.yml
```

**3. Пам'ятайте:**

- 🔒 За замовчуванням безпечно (`false`)
- ⚠️ Використовуйте тільки для dev/test
- 🚫 Ніколи не використовуйте в продакшн
- ✅ Не потрібно змінювати код модуля

---

**Потрібна допомога?** Відкрийте issue на GitHub або перегляньте повну документацію вище.
