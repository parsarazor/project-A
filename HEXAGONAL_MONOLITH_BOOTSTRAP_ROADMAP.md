# نقشه‌راه استارت از صفر تا تکمیل پروژه هگزاگونال مونولیتیک (URL Shortener)

این سند یک پلن اجرایی واقعی برای شروع پروژه از صفر است؛ هر مرحله باید کامل و تایید شود تا وارد مرحله بعدی شوید.

## فاز 0: Inception و Scope
هدف: جلوگیری از شروع مبهم و تغییرات پرهزینه بعدی.

- [ ] تعریف مسئله و ارزش محصول (چرا URL Shortener؟ برای چه کاربری؟)
- [ ] تعریف MVP (ثبت‌نام، لاگین، ساخت لینک کوتاه، ریدایرکت، پنل کاربر)
- [ ] تعریف Non-Functional Requirements اولیه:
  - [ ] امنیت (JWT، هش پسورد، rate limit)
  - [ ] کارایی (latency redirect، throughput)
  - [ ] قابلیت نگه‌داری (Hexagonal boundaries)
- [ ] تعریف معیار موفقیت نسخه اول (KPI)
- [ ] تایید نهایی Scope و عدم افزودن فیچر خارج از MVP

خروجی فاز:
- PRD کوتاه
- MVP Scope ثابت
- Definition of Done نسخه 1

## فاز 1: معماری و طراحی تکنیکی
هدف: ساخت اسکلت پایدار قبل از کدنویسی فیچرها.

- [ ] انتخاب سبک معماری: Hexagonal Monolith
- [ ] تعریف لایه‌ها:
  - [ ] Core (`internal/core/usecase`, `internal/core/ports`)
  - [ ] Adapters (`internal/adapter/*`, `internal/repository`, `internal/handler`)
  - [ ] Composition Root (`cmd/api/main.go`)
- [ ] تعریف dependency rule: فقط از بیرون به داخل
- [ ] طراحی مدل دامنه (User, URL)
- [ ] طراحی contracts/ports:
  - [ ] `UserRepository`
  - [ ] `URLRepository`
  - [ ] `TokenManager`
  - [ ] `PasswordHasher`
  - [ ] `ShortCodeGenerator`
- [ ] طراحی error strategy (دامنه، اپلیکیشن، HTTP mapping)

خروجی فاز:
- Architecture Decision Record (ADR)
- Dependency Map
- Port Contracts

## فاز 2: پایه‌های پروژه (Bootstrap)
هدف: آماده‌سازی محیط توسعه و استانداردها.

- [ ] ساخت repo و ساختار پوشه‌ها
- [ ] تنظیم Go module و dependencyهای اصلی
- [ ] تنظیم config loader (`.env`, defaults)
- [ ] تنظیم docker-compose برای DB
- [ ] تنظیم Makefile (run, test, lint, build)
- [ ] تنظیم logging و global error handler
- [ ] تنظیم migration strategy (AutoMigrate یا migration files)
- [ ] تعریف coding conventions (naming, package boundaries)

خروجی فاز:
- Skeleton قابل اجرا با health endpoint
- Developer onboarding کوتاه در README

## فاز 3: Core Use Cases (بدون HTTP/DB)
هدف: منطق کسب‌وکار در هسته، مستقل از فریم‌ورک.

- [ ] پیاده‌سازی `AuthUseCase`
  - [ ] Register
  - [ ] Login
  - [ ] ValidateToken
  - [ ] ChangePassword
  - [ ] CreateAdmin
- [ ] پیاده‌سازی `URLUseCase`
  - [ ] CreateShortURL
  - [ ] GetURLByShortCode
  - [ ] GetUserURLs
  - [ ] Delete/Activate/Deactivate
  - [ ] UserStats
- [ ] اعمال validation ruleها داخل use case
- [ ] تعریف خطاهای use case و سناریوهای business

خروجی فاز:
- Core کاملا مستقل از Fiber/GORM

## فاز 4: Secondary Adapters
هدف: اتصال Core به تکنولوژی‌ها از طریق Port.

- [ ] پیاده‌سازی repository adapter با GORM
  - [ ] UserRepositoryGorm
  - [ ] URLRepositoryGorm
- [ ] پیاده‌سازی security adapters
  - [ ] JWT manager
  - [ ] Bcrypt hasher
- [ ] پیاده‌سازی shortcode generator adapter
- [ ] mapping خطاهای ORM به خطاهای قابل فهم core

خروجی فاز:
- Adapterها با تست repository-level

## فاز 5: Primary Adapter (HTTP)
هدف: API قابل استفاده بدون نشت منطق به handler.

- [ ] طراحی endpointهای REST
- [ ] پیاده‌سازی handlerها (thin handlers)
- [ ] پیاده‌سازی middleware احراز هویت و role checks
- [ ] mapping دقیق خطاهای use case به status code
- [ ] پاسخ استاندارد JSON و schema پایدار

خروجی فاز:
- API v1 کامل برای MVP

## فاز 6: Testing Pyramid واقعی
هدف: اعتمادپذیری قبل از استقرار.

- [ ] Unit test برای use caseها (با mock ports)
- [ ] Unit test برای utilها و validatorها
- [ ] Integration test برای endpointها با SQLite/Postgres test DB
- [ ] Happy path + edge cases + failure cases
- [ ] پوشش سناریوهای امنیتی پایه
- [ ] تعریف test data builders/factories

خروجی فاز:
- تست پایدار در CI
- حداقل baseline coverage توافق‌شده تیم

## فاز 7: امنیت و پایداری عملیاتی
هدف: آماده‌سازی برای محیط واقعی.

- [ ] Secret management (عدم استفاده از default secret)
- [ ] Rate limiting برای login/redirect
- [ ] ورودی‌سنجی سخت‌گیرانه‌تر و sanitize
- [ ] CORS policy دقیق
- [ ] timeout/retry policy برای DB
- [ ] graceful shutdown
- [ ] audit log برای عملیات حساس admin

خروجی فاز:
- Security checklist پاس شده

## فاز 8: Observability
هدف: قابل عیب‌یابی شدن سیستم در production.

- [ ] Structured logging
- [ ] Request ID / Correlation ID
- [ ] متریک‌های کلیدی:
  - [ ] Redirect latency
  - [ ] Error rate
  - [ ] DB query latency
- [ ] health/readiness endpoints
- [ ] dashboard و alertهای پایه

خروجی فاز:
- حداقل مانیتورینگ عملیاتی فعال

## فاز 9: CI/CD و Release
هدف: تحویل امن و تکرارپذیر.

- [ ] CI pipeline: lint + test + build
- [ ] ساخت Docker image versioned
- [ ] اجرای migration در release process
- [ ] deployment strategy (rolling/recreate)
- [ ] rollback plan
- [ ] release notes template

خروجی فاز:
- اولین deploy پایدار در محیط staging/production

## فاز 10: Post-Launch و Iteration
هدف: رشد کنترل‌شده و حفظ کیفیت.

- [ ] بررسی KPI نسخه اول
- [ ] تحلیل bottleneckها
- [ ] backlog فنی (refactor, performance)
- [ ] backlog محصول (custom alias, analytics, expiration)
- [ ] تصمیم‌گیری برای شکستن مونولیت فقط با شواهد واقعی

خروجی فاز:
- برنامه نسخه 1.1 و 1.2

---

## ترتیب اجرایی پیشنهادی (خیلی مهم)

1. فاز 0 و 1 را کامل کن؛ بدون این دو فاز کدنویسی نکن.
2. فاز 2 را تا health endpoint بالا بیاور.
3. مستقیم برو سراغ فاز 3 (Core) و use caseها را تست‌پذیر بساز.
4. بعد فاز 4 و 5 را انجام بده تا سیستم end-to-end کار کند.
5. قبل از هر deploy، فاز 6 و 7 حداقل‌های لازم را پاس کن.
6. فاز 8 و 9 برای production اجباری‌اند.
7. فاز 10 باید دائمی باشد، نه یک‌باره.

## معیار پذیرش نهایی پروژه

- [ ] تمام endpointهای MVP کار می‌کنند
- [ ] تمام تست‌های unit/integration سبز هستند
- [ ] dependency direction نقض نشده است
- [ ] README و runbook به‌روزند
- [ ] release قابل rollback است
- [ ] حداقل observability عملیاتی فعال است

