# Phase 08 - Observability

## هدف
قابل‌دید شدن رفتار سیستم در محیط واقعی.

## چرا مهم است؟
بدون observability، رفع مشکل production حدس و گمان می‌شود.

## باید چه خروجی بدهی؟
- structured logging
- metrics کلیدی
- health/readiness endpoints

## قدم‌های اجرایی
1. برای هر request یک request-id ثبت کن.
2. لاگ‌ها را structured و قابل جستجو نگه دار.
3. latency/error-rate را متریک‌دار کن.
4. dashboard و alert حداقلی بساز.
5. runbook خطاهای رایج بنویس.

## اشتباهات رایج
- log بدون context
- نداشتن alert برای error spike
- نبودن readiness check

## مسیر تحقیق
- SLI/SLO basics
- RED metrics
- OpenTelemetry concepts
