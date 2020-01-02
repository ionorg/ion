import 'package:flutter/material.dart';
import 'package:fluro/fluro.dart';
import 'package:provider/provider.dart';
import 'page/login_page.dart';
import 'provider/client_provider.dart';
import 'router/application.dart';
import 'router/routes.dart';
import 'utils/utils.dart';

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final router = Router();
    Routes.configureRoutes(router);
    Application.router = router;

    return MultiProvider(
      providers: [
        ChangeNotifierProvider(builder: (_) => ClientProvider()),
      ],
      child: MaterialApp(
        debugShowCheckedModeBanner: false,
        title: '',
        onGenerateRoute: Application.router.generator,
        home: LoginPage(),
        theme: mDefaultTheme,
      ),
    );
  }
}

final ThemeData mDefaultTheme = ThemeData(
  primaryColor: string2Color('#0a0a0a'),
);
