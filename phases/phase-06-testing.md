# Phase 06 - Testing Pyramid

## هدف
ساخت اعتماد واقعی قبل از deploy.

## چرا مهم است؟
تست درست، سرعت توسعه را بالا می‌برد چون ریسک تغییر را کم می‌کند.

## باید چه خروجی بدهی؟
- unit tests برای core usecases
- integration tests برای API
- سناریوهای خطا + edge case

## قدم‌های اجرایی
1. unit test usecaseها با mock ports.
2. integration test endpointها با DB تست.
3. مسیرهای موفق و ناموفق را پوشش بده.
4. تست auth، role، validation، و redirect را کامل کن.
5. تست‌ها را در CI enforce کن.

## اشتباهات رایج
- فقط تست happy path
- وابستگی test به environment شخصی
- تست‌های flaky و غیرقطعی

## مسیر تحقیق
- Test doubles in Go
- Contract testing basics
- Flaky test mitigation
