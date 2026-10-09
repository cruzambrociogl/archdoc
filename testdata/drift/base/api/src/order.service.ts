import { OrderRepository } from './order.repository';

export class OrderService {
  constructor(private orderRepository: OrderRepository) {}

  list() {
    return this.orderRepository.all();
  }

  create() {
    return this.orderRepository.insert();
  }
}
