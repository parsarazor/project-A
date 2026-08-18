# Phase 03 - Core Use Cases

## هدف
پیاده‌سازی منطق کسب‌وکار مستقل از تکنولوژی.

## چرا مهم است؟
Core قلب پروژه است. اگر اینجا تمیز باشد، تغییر adapterها ارزان می‌شود.

## باید چه خروجی بدهی؟
- usecaseهای Auth و URL کامل
- validation rules مشخص
- business errors قابل پیش‌بینی

## قدم‌های اجرایی
1. سناریوهای اصلی را به usecase method تبدیل کن.
2. ورودی‌ها را validate کن.
3. فقط از Portها استفاده کن.
4. خطاهای دامنه را شفاف و محدود نگه دار.
5. ruleهای authorization را در usecase enforce کن.

## اشتباهات رایج
- انتقال business rule به handler
- نشت gorm error به لایه بالا
- انفجار تعداد خطاهای نامنسجم

## مسیر تحقیق
- Application Service vs Domain Model
- Error modeling in Go
- Validation boundary design
