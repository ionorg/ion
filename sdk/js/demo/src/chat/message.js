export default class Message {

  constructor(messageData) {
    this.id = messageData.id;
    this.message = messageData.message;
    this.senderName = messageData.senderName || undefined;
  }
}
