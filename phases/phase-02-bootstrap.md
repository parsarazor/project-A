# Phase 02 - Bootstrap پروژه

## هدف
ساخت پایه عملیاتی توسعه (config, run, build, db, tooling).

## چرا مهم است؟
پروژه بدون bootstrap استاندارد، روی دستگاه‌های مختلف قابل تکرار نیست.

## باید چه خروجی بدهی؟
- پروژه قابل اجرا با `health` endpoint
- `.env.example` کامل
- `Makefile` و Docker Compose کاربردی

## قدم‌های اجرایی
1. Go module و ساختار پوشه‌ها را بساز.
2. config loader و defaultهای امن را پیاده کن.
3. دیتابیس dev را با docker-compose بالا بیاور.
4. logging و error handler پایه را اضافه کن.
5. migration strategy را مشخص کن.

## اشتباهات رایج
- hardcode کردن secret
- نبودن `.env.example`
- نبودن دستور اجرای یکسان برای تیم

## مسیر تحقیق
- 12-Factor App basics
- Go configuration patterns
- Docker Compose for local dev
