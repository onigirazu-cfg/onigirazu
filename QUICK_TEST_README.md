# 🚀 Швидке тестування на Ubuntu

## Підготовка (одноразово)

```bash
# 1. На Ubuntu машині (172.16.246.128):
ssh usx@172.16.246.128

# Налаштуйте passwordless sudo:
echo 'usx ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/usx
sudo chmod 0440 /etc/sudoers.d/usx
exit

# 2. На вашій Mac машині:
# Скопіюйте SSH ключ:
ssh-copy-id usx@172.16.246.128

# Перевірте з'єднання:
ssh usx@172.16.246.128 "echo 'OK'"
```

## Запуск тестів

```bash
# Просто запустіть:
./run-ubuntu-test.sh
```

## Що буде протестовано

✅ 17 модулів: command, shell, file, copy, template, package, service, user, group, lineinfile, git, systemd, sysctl, cron, archive, stat, debug

✅ Факти, змінні, sudo, шаблони, cleanup

## Очікуваний результат

```
=========================================
✅ ALL TESTS PASSED!
=========================================
```

## Якщо щось не так

Дивіться детальний гайд: `UBUNTU_TESTING_GUIDE.md`

---

**Час виконання:** 3-5 хвилин
**Лог:** `/tmp/onigirazu-test-output.log`
