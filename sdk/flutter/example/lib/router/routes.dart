import 'package:fluro/fluro.dart';
import 'package:flutter/material.dart';
import 'router_handler.dart';

class Routes{

  static String root = '/';
  static String loginPage = '/login';
  static String meetingPage = '/meeting';

  static void configureRoutes(Router router){
    router.notFoundHandler =  Handler(
      handlerFunc: (BuildContext context,Map<String,List<String>> params){
        print('error::: router');
      }
    );

    router.define(loginPage, handler: meetingHandler);
    router.define(meetingPage, handler: meetingHandler);
  }
}
