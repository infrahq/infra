# Organization Request Flow

The following diagram describes the process of loading the organization from the request, and how the various scenarios are handled with respect to authentication. Note that for "is org supplied?", the organization can come from a variety of places, (the request URL, `Infra-Organization` header, and access key) and that doesn't affect the rest of the flow.

```mermaid
flowchart TD
    A[Incoming Request] --> AC

    AC{Is org supplied?}
    AC --> |No| AD{Is multi-tenant?} %% is signup enabled
    AC --> |Yes| AG[use supplied org]
    AD --> |Yes| AH[org not set]
    AD --> |No| AF[use default org]
    AH --> B
    B{endpoint requires org?}
    B --> |Yes| C{HTTP Method}
    C --> |GET|E[org is optional, handler generates fake data if missing]
    C --> |POST|AB[fail if org not set]
    B --> |No| D[Org not checked]
```
[Edit on mermaid.live](https://mermaid.live/edit#pako:eNplkcFygjAQhl9lJzPe9AU81AGx1UOrU7l0wENKFsgICU020-kA796A2GqbU5L99t__T1qWaYFsyfJKf2YlNwRxlCrwK0h2KtO1VAW84odDSydYLB4gWKdqItbtzoI2BVjXNJVEseqvlRHtXnQHQTRQtatILggVV7TqYTYDacHKQrkG_OV7heK-9Q2t731KnMUf-WHWacKiW2ybDC6UJrBIf4jRw-OoIzDn3setzHaEwsspbFGJRktFYHxkaXCMd00V3oxct9s4PsAzUqnFVJ-cP23ibjMa8hF1Q1IrXs2h5EpUaKBAhYaTl8752XvixEHmUEtr_Vuf7qQO-2PcBWGSc1kN0P-U4W_IKNlP5azE7IzCI2zOajQ1l8L_cTu0pIxKrDFlS78V3JxTlqrec67xVnAjJGnDlmQczhl3pI9fKrueL0wkeWF4fbnsvwHB3rsZ)
<!-- Keep this link in sync with the above doc -->