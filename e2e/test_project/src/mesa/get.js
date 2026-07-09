const { Client } = require('pg')
const { validator:{validateAll} } = require('indicative') 
const { getRun } = require('util-header')

module.exports.handler = async (event, context) => {
    
    context.callbackWaitsForEmptyEventLoop = false
    let run = null
    try {
        run = await getRun(event) 
        
    } catch (error) {
        console.error('Error al obtener el run del header:: ', error)
        return { statusCode: 200 , ok : false }
    }

    const client = new Client() 

    
    try {

        await validateAll( event.body, 
            { 
                'circun': 'required|number',
                'mesa': 'required|number'
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            })


        await client.connect()

         const { rows } =  await client.query(
            ` 
            SELECT  
                mesa.*,
                candidato.*,
                usuario.*,
                mesa.id AS mesa_id,
                candidato.id AS candidato_id,
                rel_candidato_acta.votos as value,
                doc."LINK" as link,
                doc."COD_MESA" as cod_mesa 
            FROM  
                mesa
            JOIN "Documentos" doc ON ((mesa.circunscripcionid + 7000)  = doc."COD_CIRC_ELECTORAL") and (mesa.mesa = doc."NUMERO_MESA")
            JOIN candidato ON CONCAT('CIRCUNSCRIPCIÓN SENATORIAL ', mesa.cs) = candidato.territorio 
            JOIN usuario ON usuario.id = mesa.usuario_id
            FULL JOIN rel_candidato_acta ON mesa."id" = rel_candidato_acta.id_acta AND candidato."id" = rel_candidato_acta.id_candidato
            WHERE 
                mesa.circunscripcionid = $1
                AND mesa.mesa = $2

                order by candidato.voto ASC
            `, [ event.body.circun, event.body.mesa ])   

/*             const { rows } =  await client.query(
                ` 
                insert into 

                rel_candidato_acta set( id_candidato, id_acta_ votos ) values( $1,$2,$3

                )
                
                
                `, [1,1,1 ])  */

        await client.end()

        console.log(run, event.body.circun, event.body.mesa )

        return { statusCode: 200 , data : rows }
    }catch(error){
        console.error('Fallo la consulta::', error)
        if(client)
        {
            await client.end()

        }

        return { statusCode: 200 , ok : false }

    }
}