import 'package:fluro/fluro.dart';
import 'package:flutter/material.dart';
import '../page/meeting_page.dart';
import '../page/login_page.dart';

Handler meetingHandler = Handler(
  handlerFunc: (BuildContext context,Map<String,List<String>> params){
    return MeetingPage();
  }
);

Handler loginHandler = Handler(
    handlerFunc: (BuildContext context,Map<String,List<String>> params){
      return LoginPage();
    }
);
