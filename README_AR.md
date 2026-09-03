# UQDA Stack

**SOCKS5 وتحويل TCP/UDP لشبكة UQDA المشفرة من دون TUN أو root**

[English](README.md) · **العربية**

يسمح UQDA Stack للتطبيقات المختارة بالدخول إلى شبكة UQDA من دون إنشاء واجهة
شبكة افتراضية ومن دون صلاحيات مدير النظام. يوفّر SOCKS5 محليًا، وتحويل منافذ
TCP وUDP بالاتجاهين، وحل أسماء المفاتيح العامة، ويستخدم المحرك المحصّن من
[UQDA Core](https://github.com/Uqda/Core).

المشروع مشتق من [Yggstack](https://github.com/yggdrasil-network/yggstack)،
وتفاصيل النسب والترخيص موجودة في [NOTICE.md](NOTICE.md).

## البداية السريعة

```sh
mkdir -p "$HOME/.config/uqda-stack"
uqda-stack -genconf > "$HOME/.config/uqda-stack/uqda.conf"
$EDITOR "$HOME/.config/uqda-stack/uqda.conf"

uqda-stack \
  -useconffile "$HOME/.config/uqda-stack/uqda.conf" \
  -socks 127.0.0.1:1080
```

اختبار تطبيق عبر الـproxy:

```sh
curl --proxy socks5h://127.0.0.1:1080 \
  'http://[عنوان-UQDA]:8080/'
```

استخدم `socks5h` حتى يجري حل اسم النطاق من خلال الـproxy أيضًا.

## تحويل المنافذ

من منفذ محلي إلى خدمة UQDA بعيدة:

```sh
uqda-stack -useconffile uqda.conf \
  -local-tcp '127.0.0.1:8080:[عنوان-UQDA]:80'
```

من عنوان UQDA للعقدة إلى خدمة محلية محددة:

```sh
uqda-stack -useconffile uqda.conf \
  -remote-tcp '80:127.0.0.1:8080'
```

تتوفر الخيارات نفسها باسم `-local-udp` و`-remote-udp` لـUDP.

## الحماية الافتراضية

- يقبل SOCKS افتراضيًا `127.0.0.1` و`::1` وUnix socket فقط.
- يرفض `0.0.0.0` و`::` وعناوين LAN لمنع Open Proxy غير مقصود.
- يحتاج الاستماع العام إلى `-allow-public-socks` وتحذير واضح.
- يُنشأ Unix socket بصلاحية `0600`.
- ترتبط تحويلات المنافذ المحلية بـ`127.0.0.1` افتراضيًا.
- يُعطّل منفذ الإدارة داخل Stack لتجنب التعارض مع UQDA Core.

لا تستخدم المفتاح الخاص نفسه في Core وStack بالتزامن. المشروع شبكة overlay
مشفرة وليس نظام إخفاء هوية، ولم يحصل بعد على تدقيق أمني مستقل.

## البناء والاختبار

يتطلب Go 1.25.13 أو أحدث:

```sh
./build
go vet ./...
go test ./...
go test -race ./...
```

راجع [README.md](README.md) للتوثيق الكامل و[SECURITY.md](SECURITY.md) لحدود الأمان.
