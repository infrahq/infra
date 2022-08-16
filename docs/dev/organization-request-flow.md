# Organization Request Flow

The following diagram describes the process of loading the organization from the request, and how the various scenarios are handled with respect to authentication. Note that for "is org supplied?", the organization can come from a variety of places, (the request URL, `Infra-Organization` header, and access key) and that doesn't affect the rest of the flow.

```mermaid
flowchart TD
    A[Incoming Request] --> AC

    subgraph Org Selection
        AC{Is org supplied?}
        AC --> |No| AD{Is multi-tenant?} %% is signup enabled
        AC --> |Yes| AG[use supplied org]
        AD --> |Yes| AH[org not set]
        AD --> |No| AF[use default org]
        AF --> AI
        AH --> AI
        AG --> AI
        AI(continue)
    end

    subgraph Authentication and org check
        AI --> AA
        AA{Endpoint requires authentication?}
        AA --> |Yes| AB[fail if org not set]
        AA --> |No| B{endpoint requires org?}
        B --> |Yes| C{Does endpoint supply fake data <br>when there is no org supplied?}
        C --> |Yes|E[org is optional, handler generates fake data if missing]
        C --> |No|AB
        B --> |No| D[Org not checked]
    end
```
[Edit on mermaid.live](https://mermaid.live/edit#pako:eNp1U8tu2zAQ_JUFgQAtYP-AUSSgLSfxpQGaXgrJB5pcSYQpUuUDQaD437uSXYepGp2I5ezs7Aw1MOkUshWrjXuRrfARfhaVBfp4ubPSddo28AN_JwxxD8vlLfBNZc-IkA6NF30LT76BZzQoo3aXu4lhM-wCOLoMqe-NRnV3ym8nurfv7g14MSK7ZKJeRrTCxrsT3NyADhB0Y1MPVDwYVPP2Xxio_6FMAa9jxpn7DFrk0MdyVGRdhIDxP6hJz_3Ep7AWpOlfuvuzD7us9DgvPcxLuy_S2ahtwq_nKlo1M5On2CKhpBjdBGGndUC2KI8515meZyU-bK3qnbYRPCWmPQYQH9g--M9zV9ZlLbQBXcMn7vB3d9YDzuZQV06-zrg3Q-EIce2ZUnqFWhzJYREFfDv42xdSCSTV4xi6dZ8-myz27ZQkwV0_LifMAlqyy6CHBi16EWns-xjardMh0IPez-hoLb6eyR-XLcqniyFTAqj21-jYgnXoO6EV_UDDWK4YrdBhxVZ0VMIfK1bZE-FSTxJwq3R0nq1qYQIuGGXjnl-tZKvoE_4FFVrQQ-guqNMfvU4d_A)
<!-- Keep this link in sync with the above doc -->