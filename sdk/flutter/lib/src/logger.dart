class Logger {
  String _app_name;
  Logger(this._app_name) {}

  void error(error) {
    print('[' + _app_name + '] ERROR: ' + error);
  }

  void debug(msg) {
    print('[' + _app_name + '] DEBUG: ' + msg);
  }

  void warn(msg) {
    print('[' + _app_name + '] WARN: ' + msg);
  }

  void failure(error) {
    var log = '[' + _app_name + '] FAILURE: ' + error;
    print(log);
    throw (log);
  }
}
