import { CustomerTable } from './customer.table';

@Table({ name: 'orders' })
export class OrderTable {
  @PrimaryGeneratedColumn()
  id!: string;

  @ForeignKeyColumn(() => CustomerTable)
  customerId!: string;

  @Column({ type: 'numeric' })
  total!: number;
}
