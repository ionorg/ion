#!/bin/bash

FLUTTER_APP_FOLDER=$(cd `dirname $0`/../; pwd)
FLUTTER_APP_ORG=com.github.pion
FLUTTER_APP_PROJECT_NAME=ion_sfu_example
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
        flutter create --project-name $FLUTTER_APP_PROJECT_NAME --org $FLUTTER_APP_ORG .
        add_permission_label
    else
        echo "Project [$FLUTTER_APP_PROJECT_NAME] already exists!"
    fi
}

function add_permission_label() {
    cd $FLUTTER_APP_FOLDER/scripts
    echo ""
    echo "Add permission label for iOS."
    echo ""
    python add-line.py -i ../ios/Runner/Info.plist -n 25 -t '	<key>NSCameraUsageDescription</key>'
    python add-line.py -i ../ios/Runner/Info.plist -n 26 -t '	<string>$(PRODUCT_NAME) Camera Usage!</string>'
    python add-line.py -i ../ios/Runner/Info.plist -n 27 -t '	<key>NSMicrophoneUsageDescription</key>'
    python add-line.py -i ../ios/Runner/Info.plist -n 28 -t '	<string>$(PRODUCT_NAME) Microphone Usage!</string>'
    python add-line.py -i ../ios/Podfile -n 3 -t 'platform :ios, '9.0''
    echo ""
    echo "Add permission label for Android."
    echo ""
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 9 -t '    <uses-feature android:name="android.hardware.camera" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 10 -t '    <uses-feature android:name="android.hardware.camera.autofocus" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 11 -t '    <uses-permission android:name="android.permission.CAMERA" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 12 -t '    <uses-permission android:name="android.permission.RECORD_AUDIO" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 13 -t '    <uses-permission android:name="android.permission.WAKE_LOCK" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 14 -t '    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 15 -t '    <uses-permission android:name="android.permission.CHANGE_NETWORK_STATE" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 16 -t '    <uses-permission android:name="android.permission.MODIFY_AUDIO_SETTINGS" />'
    python add-line.py -i ../android/app/src/main/AndroidManifest.xml -n 17 -t '    <uses-permission android:name="android.permission.WRITE_EXTERNAL_STORAGE" />'
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
