# Phase 05 - Primary Adapter (HTTP)

## هدف
ارائه API تمیز و نازک که فقط orchestration انجام دهد.

## چرا مهم است؟
Handler باید ساده بماند تا maintainability و تست endpoint بالا برود.

## باید چه خروجی بدهی؟
- endpointهای v1 پایدار
- middleware auth/role
- mapping خطاها به status code درست

## قدم‌های اجرایی
1. routeها را version‌بندی کن (`/api/v1`).
2. request parsing و response shaping را در handler نگه دار.
3. منطق اصلی را به usecase پاس بده.
4. middleware authentication و authorization اضافه کن.
5. contract پاسخ JSON را ثابت نگه دار.

## اشتباهات رایج
- business rule در handler
- response schema ناپایدار
- status code اشتباه (مثلا 500 به‌جای 401/403/422)

## مسیر تحقیق
- REST API design conventions
- Idempotency and HTTP semantics
- Error mapping patterns
