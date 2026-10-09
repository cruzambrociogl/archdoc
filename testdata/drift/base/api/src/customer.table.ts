@Table({ name: 'customers' })
export class CustomerTable {
  @PrimaryGeneratedColumn()
  id!: string;

  @Column({ type: 'text' })
  email!: string;
}
