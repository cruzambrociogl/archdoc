export class OrderRepository {
  all() {
    return this.db.selectFrom('orders').selectAll().execute();
  }

  insert() {
    return this.db.insertInto('orders').values({}).execute();
  }
}
