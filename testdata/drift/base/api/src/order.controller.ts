import { OrderService } from './order.service';

@Controller('orders')
export class OrderController {
  constructor(private service: OrderService) {}

  @Get()
  list() {
    return this.service.list();
  }

  @Post()
  create() {
    return this.service.create();
  }
}
