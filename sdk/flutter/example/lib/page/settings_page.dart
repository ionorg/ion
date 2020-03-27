import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../utils/utils.dart';

class SettingsPage extends StatefulWidget {
  SettingsPage({Key key}) : super(key: key);
  @override
  _MySettingsPage createState() => _MySettingsPage();
}

class _MySettingsPage extends State<SettingsPage> {
  String _resolution;
  String _bandwidth;
  String _codec;
  String _displayName;

  SharedPreferences _preferences;
  @override
  initState() {
    super.initState();
    _loadSettings();
  }

  @override
  deactivate() {
    super.deactivate();
    _saveSettings();
  }

  void _loadSettings() async {
    _preferences = await SharedPreferences.getInstance();
    this.setState(() {
      _resolution = _preferences.getString('resolution') ?? 'vga';
      _bandwidth = _preferences.getString('bandwidth') ?? '512';
      _displayName = _preferences.getString('display_name') ?? 'Guest';
      _codec = _preferences.getString('codec') ?? 'vp8';
    });
  }

  void _saveSettings() {
    _preferences.setString('resolution', _resolution);
    _preferences.setString('bandwidth', _bandwidth);
    _preferences.setString('display_name', _displayName);
    _preferences.setString('codec', _codec);
  }

  void _handleSave(BuildContext context) {
    Navigator.of(context).pop();
  }

  var _codecItems = [
    {
      'name': 'H264',
      'value': 'h264',
    },
    {
      'name': 'VP8',
      'value': 'vp8',
    },
    {
      'name': 'VP9',
      'value': 'VP9',
    },
  ];

  _onCodecChanged(String value) {
    setState(() {
      _codec = value;
    });
  }

  var _bandwidthItems = [
    {
      'name': '256kbps',
      'value': '256',
    },
    {
      'name': '512kbps',
      'value': '512',
    },
    {
      'name': '768kbps',
      'value': '768',
    },
    {
      'name': '1Mbps',
      'value': '1024',
    },
  ];

  _onbandwidthChanged(String value) {
    setState(() {
      _bandwidth = value;
    });
  }

  var _resolutionItems = [
    {
      'name': 'QVGA',
      'value': 'qvga',
    },
    {
      'name': 'VGA',
      'value': 'vga',
    },
    {
      'name': 'HD',
      'value': 'hd',
    },
  ];

  _onResolutionChanged(String value) {
    setState(() {
      _resolution = value;
    });
  }

  Widget _buildRowFixTitleRadio(List<Map<String, dynamic>> items, var value,
      ValueChanged<String> onValueChanged) {
    return Container(
        width: 320,
        height: 100,
        child: GridView.count(
            crossAxisCount: 2,
            crossAxisSpacing: 10.0,
            childAspectRatio: 2.8,
            children: items
                .map((item) => ConstrainedBox(
                      constraints:
                          BoxConstraints.tightFor(width: 120.0, height: 36.0),
                      child: RadioListTile<String>(
                        value: item['value'],
                        title: Text(item['name']),
                        groupValue: value,
                        onChanged: (value) => onValueChanged(value),
                      ),
                    ))
                .toList()));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        appBar: AppBar(
          title: Text("Settings"),
        ),
        body: Align(
            alignment: Alignment(0, 0),
            child: SingleChildScrollView(
              padding: EdgeInsets.all(16.0),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  mainAxisAlignment: MainAxisAlignment.start,
                  children: <Widget>[
                    Column(
                      children: <Widget>[
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(46.0, 18.0, 48.0, 0),
                          child: Align(
                            child: Text('DisplayName:'),
                            alignment: Alignment.centerLeft,
                          ),
                        ),
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(48.0, 0.0, 48.0, 0),
                          child: TextField(
                            keyboardType: TextInputType.text,
                            textAlign: TextAlign.center,
                            decoration: InputDecoration(
                              contentPadding: EdgeInsets.all(10.0),
                              border: UnderlineInputBorder(
                                  borderSide:
                                      BorderSide(color: Colors.black12)),
                              hintText: _displayName,
                            ),
                            onChanged: (value) {
                              setState(() {
                                _displayName = value;
                              });
                            },
                          ),
                        ),
                      ],
                    ),
                    Column(
                      children: <Widget>[
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(46.0, 18.0, 48.0, 0),
                          child: Align(
                            child: Text('Codec:'),
                            alignment: Alignment.centerLeft,
                          ),
                        ),
                        Padding(
                          padding: const EdgeInsets.fromLTRB(4.0, 0.0, 4.0, 0),
                          child: _buildRowFixTitleRadio(
                              _codecItems, _codec, _onCodecChanged),
                        ),
                      ],
                    ),
                    Column(
                      children: <Widget>[
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(46.0, 18.0, 48.0, 0),
                          child: Align(
                            child: Text('Resolution:'),
                            alignment: Alignment.centerLeft,
                          ),
                        ),
                        Padding(
                          padding: const EdgeInsets.fromLTRB(4.0, 0.0, 4.0, 0),
                          child: _buildRowFixTitleRadio(_resolutionItems,
                              _resolution, _onResolutionChanged),
                        ),
                      ],
                    ),
                    Column(
                      children: <Widget>[
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(46.0, 18.0, 48.0, 0),
                          child: Align(
                            child: Text('Bandwidth:'),
                            alignment: Alignment.centerLeft,
                          ),
                        ),
                        Padding(
                          padding: const EdgeInsets.fromLTRB(4.0, 0.0, 4.0, 0),
                          child: _buildRowFixTitleRadio(
                              _bandwidthItems, _bandwidth, _onbandwidthChanged),
                        ),
                      ],
                    ),
                    Padding(
                        padding: const EdgeInsets.fromLTRB(0.0, 18.0, 0.0, 0.0),
                        child: Container(
                            height: 48.0,
                            width: 160.0,
                            child: InkWell(
                              child: Container(
                                width: 220.0,
                                height: 48.0,
                                decoration: BoxDecoration(
                                  border: Border.all(
                                    color: string2Color('#e13b3f'),
                                    width: 1,
                                  ),
                                ),
                                child: Center(
                                  child: Text(
                                    'Save',
                                    style: TextStyle(
                                      fontSize: 16.0,
                                      color: Colors.black,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                ),
                              ),
                              onTap: () => _handleSave(context),
                            )))
                  ]),
            )));
  }
}
