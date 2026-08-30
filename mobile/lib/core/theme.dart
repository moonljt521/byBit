import 'package:flutter/material.dart';

/// 交易所风格暗色主题。
class AppTheme {
  AppTheme._();

  static const up = Color(0xFF0ECB81);
  static const down = Color(0xFFF6465D);
  static const accent = Color(0xFF00C8A0);
  static const bg = Color(0xFF0B0E11);
  static const surface = Color(0xFF161A1E);
  static const border = Color(0xFF252930);

  static ThemeData dark() {
    final base = ThemeData.dark(useMaterial3: true);
    return base.copyWith(
      scaffoldBackgroundColor: bg,
      colorScheme: base.colorScheme.copyWith(
        primary: accent,
        secondary: accent,
        surface: surface,
        error: down,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: bg,
        elevation: 0,
        centerTitle: false,
      ),
      cardTheme: CardTheme(
        color: surface,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(10),
          side: const BorderSide(color: border),
        ),
        margin: EdgeInsets.zero,
      ),
      dividerColor: border,
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xFF1E2329),
        hintStyle: const TextStyle(color: Color(0x38FFFFFF), fontSize: 13), // 极浅占位符
        labelStyle: const TextStyle(color: Colors.white38, fontSize: 13),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: border),
        ),
        isDense: true,
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: surface,
        indicatorColor: accent.withOpacity(0.18),
        labelTextStyle: WidgetStateProperty.all(
          const TextStyle(fontSize: 12),
        ),
      ),
      snackBarTheme: const SnackBarThemeData(behavior: SnackBarBehavior.floating),
    );
  }
}
