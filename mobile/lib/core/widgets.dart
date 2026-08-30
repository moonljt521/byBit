import 'package:flutter/material.dart';

import 'models.dart';
import 'theme.dart';

/// 蜡烛图（CustomPaint 自绘：价格主图 + 成交量副图，无第三方图表依赖）。
class CandleChart extends StatelessWidget {
  const CandleChart({super.key, required this.candles, this.height = 240});

  final List<Candle> candles;
  final double height;

  @override
  Widget build(BuildContext context) {
    final shown = candles.length > 120 ? candles.sublist(candles.length - 120) : candles;
    return LayoutBuilder(
      builder: (context, constraints) => CustomPaint(
        size: Size(constraints.maxWidth, height),
        painter: _CandlePainter(shown),
      ),
    );
  }
}

class _CandlePainter extends CustomPainter {
  _CandlePainter(this.candles);

  final List<Candle> candles;

  @override
  void paint(Canvas canvas, Size size) {
    if (candles.isEmpty) return;
    var minL = candles.first.low;
    var maxH = candles.first.high;
    var maxVol = 0.0;
    for (final c in candles) {
      if (c.low < minL) minL = c.low;
      if (c.high > maxH) maxH = c.high;
      if (c.volume > maxVol) maxVol = c.volume;
    }
    final pad = (maxH - minL) * 0.05 + 0.0001;
    minL -= pad;
    maxH += pad;

    final chartH = size.height * 0.8;
    final volTop = size.height * 0.85;
    final volH = size.height * 0.13;
    final w = size.width / candles.length;
    final bodyW = w * 0.66;
    final paint = Paint()..strokeWidth = 1;
    final grid = Paint()
      ..color = AppTheme.border
      ..strokeWidth = 0.5;

    for (var i = 1; i <= 4; i++) {
      final y = chartH * i / 5;
      canvas.drawLine(Offset(0, y), Offset(size.width, y), grid);
    }

    for (var i = 0; i < candles.length; i++) {
      final c = candles[i];
      final x = w * i + w / 2;
      double y(double p) => chartH * (1 - (p - minL) / (maxH - minL));
      final color = c.up ? AppTheme.up : AppTheme.down;
      paint.color = color;

      canvas.drawLine(Offset(x, y(c.high)), Offset(x, y(c.low)), paint);
      final top = y(c.open > c.close ? c.open : c.close);
      final bottom = y(c.open > c.close ? c.close : c.open);
      canvas.drawRect(
        Rect.fromLTRB(x - bodyW / 2, top, x + bodyW / 2, bottom == top ? top + 1 : bottom),
        paint,
      );
      final vh = maxVol > 0 ? c.volume / maxVol * volH : 0.0;
      canvas.drawRect(
        Rect.fromLTRB(x - bodyW / 2, volTop + volH - vh, x + bodyW / 2, volTop + volH),
        paint..color = color.withOpacity(0.55),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _CandlePainter oldDelegate) => oldDelegate.candles != candles;
}
