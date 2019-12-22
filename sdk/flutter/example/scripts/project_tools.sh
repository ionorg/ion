#!/bin/bash

FLUTTER_APP_FOLDER=$(cd `dirname $0`/../; pwd)
FLUTTER_APP_ORG=com.github.pion.ion
FLUTTER_APP_PROJECT_NAME=ion_flutter_example
CMD=$1

function cleanup() {
    echo "Cleanup project [$FLUTTER_APP_PROJECT_NAME] files ..."
    cd $FLUTTER_APP_FOLDER
    rm -rf android build *.iml ios pubspec.lock test .flutter-plugins .metadata .packages .idea
}

function create() {
    cd $FLUTTER_APP_FOLDER
    if [ ! -d "ios" ] && [ ! -d "android" ];then
        echo "Create flutter project: name=$FLUTTER_APP_PROJECT_NAME, org=$FLUTTER_APP_ORG ..."
        flutter create --android-language java --ios-language objc --project-name $FLUTTER_APP_PROJECT_NAME --org $FLUTTER_APP_ORG .
        add_permission_label
    else
        echo "Project [$FLUTTER_APP_PROJECT_NAME] already exists!"
    fi
}

function add_permission_label() {
    cd $FLUTTER_APP_FOLDER/scripts
    echo ""
    echo "Add permission labels to Info.plist."
    echo ""
    python add-line.py -i ../ios/Runner/Info.plist -s '<key>UILaunchStoryboardName</key>' -t '	<key>NSCameraUsageDescription</key>'
    python add-line.py -i ../ios/Runner/Info.plist -s '<key>UILaunchStoryboardName</key>' -t '	<string>$(PRODUCT_NAME) Camera Usage!</string>'
    python add-line.py -i ../ios/Runner/Info.plist -s '<key>UILaunchStoryboardName</key>' -t '	<key>NSMicrophoneUsageDescription</key>'
    python add-line.py -i ../ios/Runner/Info.plist -s '<key>UILaunchStoryboardName</key>' -t '	<string>$(PRODUCT_NAME) Microphone Usage!</string>'
    python add-line.py -i ../ios/Podfile -s "# platform :ios" -t "platform :ios" -r
    echo ""
    echo "Add permission labels to AndroidManifest.xml."
    echo ""
    python add-line.py -i ../android/app/build.gradle -s 'minSdkVersion 16' -t 'minSdkVersion 18' -r
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.CAMERA" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.RECORD_AUDIO" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.WAKE_LOCK" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.CHANGE_NETWORK_STATE" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.MODIFY_AUDIO_SETTINGS" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -s "<application" -t '    <uses-permission android:name="android.permission.WRITE_EXTERNAL_STORAGE" />'
}

if [ "$CMD" == "create" ];
then
    create
fi

if [ "$CMD" == "cleanup" ];
then
    cleanup
fi

if [ "$CMD" == "add_permission" ];
then
    add_permission_label
fi

if [ ! -n "$1" ] ;then
    echo "Usage: ./project_tools.sh 'create' | 'cleanup'"
fi
