# Phase 01 - Architecture (Hexagonal Monolith)

## هدف
طراحی مرزهای درست بین Core و Adapters.

## چرا مهم است؟
این فاز هزینه نگهداری آینده را تعیین می‌کند. مرز درست یعنی تغییر DB/HTTP با کمترین درد.

## باید چه خروجی بدهی؟
- ADR معماری
- نمودار dependency direction
- تعریف Portها

## قدم‌های اجرایی
1. Core را تعریف کن: `usecase` و `ports`.
2. Adapterها را دسته‌بندی کن: primary (HTTP) و secondary (DB/JWT/Bcrypt).
3. قانون وابستگی را ثبت کن: فقط بیرون -> داخل.
4. برای هر نیاز Core، یک Port اینترفیس بنویس.

## اشتباهات رایج
- تزریق مستقیم GORM/Fiber به usecase
- قاطی کردن response/http model با domain logic
- نداشتن composition root شفاف

## مسیر تحقیق
- Ports and Adapters Architecture
- Dependency Inversion Principle
- Composition Root in Go
